package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"
)

// NotificationWorker mengerjakan task "notification:send".
//
// Bagi worker, Asynq adalah framework-nya — sama seperti Fiber bagi API.
// Karena itu method di sini menerima *asynq.Task, seperti method service lain
// menerima *fiber.Ctx.
type NotificationWorker struct {
	users         *repository.UserRepository
	notifications *repository.NotificationRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	users *repository.UserRepository,
	notifications *repository.NotificationRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{users: users, notifications: notifications,
		tx: tx, mailer: mailer, logger: logger}
}

// HandleSend mengembalikan error biasa untuk meminta Asynq melakukan retry
// nanti, atau error yang membungkus asynq.SkipRetry untuk menyatakan "jangan
// di-retry, task ini tidak mungkin berhasil".
//
// Asynq menjamin setiap task dijalankan PALING SEDIKIT satu kali, bukan TEPAT
// satu kali. Task yang sama bisa datang dua kali (mis. worker mati setelah
// mengirim email tetapi sebelum sempat melapor ke Redis). Karena itu handler
// ini harus idempoten: menjalankannya dua kali untuk task yang sama harus
// berakibat sama dengan menjalankannya sekali.
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}
	var event model.EnrollmentEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	// Satu event course.quota_changed menotifikasi BANYAK peserta; event
	// enrollment menotifikasi satu mahasiswa saja.
	if job.EventType == model.EventCourseQuotaChanged {
		return w.handleQuotaChanged(ctx, job, event)
	}
	return w.handleEnrollment(ctx, job, event)
}

// handleEnrollment menotifikasi satu mahasiswa (enroll/drop/swap).
func (w *NotificationWorker) handleEnrollment(ctx context.Context, job model.NotificationJob, event model.EnrollmentEvent) error {
	user, err := w.users.FindByID(ctx, event.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		// Akunnya sudah dihapus. Melakukan retry tidak akan memunculkannya.
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
	}
	if err != nil {
		return err // database sedang bermasalah: layak di-retry
	}

	msg, err := BuildMessage(job.EventType, event, user)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	alreadySent := false
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		// Claim dan pengiriman berada di dalam SATU transaction:
		//   - pengiriman gagal  -> claim ikut dibatalkan -> attempt berikutnya
		//     masih dapat mengirim;
		//   - pengiriman sukses -> claim di-commit -> attempt berikutnya
		//     melihat claim ini dan tidak mengirim lagi.
		claimed, err := w.notifications.Claim(ctx, model.Notification{
			OutboxID: job.OutboxID, UserID: user.ID, Channel: "email",
			Subject: msg.Subject, Body: msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}
		return w.mailer.Send(ctx, msg)
	})

	// SkipRetry HARUS dibungkus dengan %w agar errors.Is di dalam Asynq
	// mengenalinya. Kegagalan permanen langsung masuk archived, tidak di-retry.
	if errors.Is(err, helper.ErrPermanentDelivery) {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", alreadySent))
	return nil
}

// handleQuotaChanged menotifikasi SELURUH peserta kelas untuk satu event
// perubahan kuota (Tugas Mandiri D.2).
//
// Setiap peserta di-claim dan dikirimi email dalam transaction-nya SENDIRI,
// memakai idempotency key (outbox_id, user_id). Konsekuensinya, bila email ke-7
// dari 40 gagal, task di-retry dan hanya peserta yang BELUM ter-claim yang
// dikirim ulang — peserta 1..6 dilewati karena claim-nya sudah commit.
func (w *NotificationWorker) handleQuotaChanged(ctx context.Context, job model.NotificationJob, event model.EnrollmentEvent) error {
	retry, _ := asynq.GetRetryCount(ctx)
	var pending error // kegagalan sementara/DB yang menandakan task perlu di-retry

	for _, uid := range event.Participants {
		user, err := w.users.FindByID(ctx, uid)
		if errors.Is(err, repository.ErrNotFound) {
			continue // akun peserta sudah dihapus: lewati
		}
		if err != nil {
			return err // database bermasalah: retry seluruh task
		}

		msg, err := BuildMessage(job.EventType, event, user)
		if err != nil {
			return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}

		alreadySent := false
		sendErr := w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
			claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
				OutboxID: job.OutboxID, UserID: user.ID, Channel: "email",
				Subject: msg.Subject, Body: msg.Body,
			})
			if err != nil {
				return err
			}
			if !claimed {
				alreadySent = true
				return nil
			}
			return w.mailer.Send(ctx, msg)
		})

		if errors.Is(sendErr, helper.ErrPermanentDelivery) {
			// Alamat satu peserta permanen gagal: lewati peserta itu saja,
			// jangan sampai menghentikan pengiriman ke peserta lain atau
			// membuat seluruh task archived.
			w.logger.Warn("notification_skipped_permanent",
				slog.Int64("outbox_id", job.OutboxID),
				slog.Int("user_id", user.ID),
				slog.String("error", sendErr.Error()))
			continue
		}
		if sendErr != nil {
			// Temporary/DB: ingat lalu lanjut ke peserta lain. Task akan
			// di-retry; peserta yang sudah ter-claim dilewati, hanya yang
			// gagal yang diulang.
			pending = sendErr
			continue
		}

		w.logger.Info("notification_processed",
			slog.Int64("outbox_id", job.OutboxID),
			slog.String("event", job.EventType),
			slog.Int("user_id", user.ID),
			slog.Int("retry", retry),
			slog.Bool("already_sent", alreadySent))
	}
	return pending
}
