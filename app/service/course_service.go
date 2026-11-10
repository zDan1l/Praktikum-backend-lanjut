package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"
)

// CourseService melayani endpoint KRS: mendaftar, keluar, dan pindah kelas.
// Setiap perubahan KRS dicatat ke outbox DI DALAM transaction yang sama,
// sehingga notifikasi tidak pernah dibuat untuk pendaftaran yang batal dan
// tidak pernah hilang ketika Redis mati (lihat Modul 11, A.3).
type CourseService struct {
	courses *repository.CourseRepository
	outbox  *repository.OutboxRepository
	tx      *repository.TxManager
}

func NewCourseService(
	courses *repository.CourseRepository,
	outbox *repository.OutboxRepository,
	tx *repository.TxManager,
) *CourseService {
	return &CourseService{courses: courses, outbox: outbox, tx: tx}
}

func conflict(code, message string) *helper.AppError {
	return &helper.AppError{Status: fiber.StatusConflict, Code: code, Message: message}
}

// ---------- POST /courses/:id/enroll ----------

func (s *CourseService) Enroll(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas tidak valid")
	}

	var enrolled model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		// 1. Ambil satu kursi secara atomik (row kelas di-lock).
		var err error
		enrolled, err = repo.IncrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}

		// 2. Catat event ke outbox memakai transaction yang SAMA. Bila
		//    langkah 3 gagal (sudah terdaftar), catatan ini ikut dibatalkan
		//    bersama penambahan kursi — tidak ada notifikasi untuk KRS batal.
		if err := s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   enrolled.ID,
			CourseCode: enrolled.Code,
			CourseName: enrolled.Name,
		}); err != nil {
			return err
		}

		// 3. Catat pendaftarannya.
		return repo.InsertEnrollment(ctx, courseID, current.UserID)
	})

	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas tidak ditemukan")
	case errors.Is(err, repository.ErrCourseFull):
		return conflict("COURSE_FULL", "kelas sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return conflict("ALREADY_ENROLLED", "Anda sudah terdaftar di kelas ini")
	case err != nil:
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusCreated, "berhasil terdaftar di kelas", enrolled)
}

// ---------- DELETE /courses/:id/enroll ----------

func (s *CourseService) Drop(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas tidak valid")
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		if err := repo.DeleteEnrollment(ctx, courseID, current.UserID); err != nil {
			return err
		}
		course, err := repo.DecrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentDropped, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   course.ID,
			CourseCode: course.Code,
			CourseName: course.Name,
		})
	})

	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("Anda tidak terdaftar di kelas ini")
	case err != nil:
		return helper.Internal(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---------- POST /enrollments/swap ----------

func (s *CourseService) Swap(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.SwapRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body tidak valid")
	}
	if req.FromCourseID < 1 || req.ToCourseID < 1 {
		return helper.BadRequest("from_course_id dan to_course_id wajib diisi")
	}
	if req.FromCourseID == req.ToCourseID {
		return helper.BadRequest("kelas asal dan tujuan tidak boleh sama")
	}

	var to model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		// Keluar dari kelas asal.
		if err := repo.DeleteEnrollment(ctx, req.FromCourseID, current.UserID); err != nil {
			return err
		}
		from, err := repo.DecrementEnrolled(ctx, req.FromCourseID)
		if err != nil {
			return err
		}

		// Masuk ke kelas tujuan.
		to, err = repo.IncrementEnrolled(ctx, req.ToCourseID)
		if err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, req.ToCourseID, current.UserID); err != nil {
			return err
		}

		// Satu event "pindah", bukan dua event "keluar" dan "masuk":
		// mahasiswa melakukan satu tindakan, maka ia menerima satu notifikasi.
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentSwapped, model.EnrollmentEvent{
			UserID:         current.UserID,
			CourseID:       to.ID,
			CourseCode:     to.Code,
			CourseName:     to.Name,
			FromCourseCode: from.Code,
			FromCourseName: from.Name,
		})
	})

	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas tidak ditemukan atau Anda tidak terdaftar di kelas asal")
	case errors.Is(err, repository.ErrCourseFull):
		return conflict("COURSE_FULL", "kelas tujuan sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return conflict("ALREADY_ENROLLED", "Anda sudah terdaftar di kelas tujuan")
	case err != nil:
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "berhasil pindah kelas", to)
}

// ---------- PATCH /courses/:id ----------
//
// Tugas mandiri D.2. UpdateQuota kini DUA pernyataan SQL: mengubah kuota dan
// mencatat event course.quota_changed ke outbox. Keduanya WAJIB dalam satu
// transaction — kalau tidak, kuota bisa ter-commit tetapi catatan event gagal
// (atau sebaliknya), sehingga peserta tidak diberi tahu atau diberi tahu untuk
// perubahan yang batal. Itulah alasan baseline satu-statement harus dibungkus
// WithTx begitu outbox ikut ditulis.
//
// Satu event menghasilkan BANYAK notifikasi (satu per peserta). Daftar peserta
// diambil sebagai snapshot di dalam transaction yang sama dan disalin ke event,
// sehingga worker tidak perlu membaca ulang keadaan kelas yang mungkin sudah
// berubah saat task dikerjakan.
func (s *CourseService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas tidak valid")
	}
	var req model.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body tidak valid")
	}
	if req.Capacity < 0 {
		return helper.BadRequest("capacity tidak boleh negatif")
	}

	var course model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		var err error
		course, err = repo.UpdateQuota(ctx, courseID, req.Capacity)
		if err != nil {
			return err
		}

		participants, err := repo.ListParticipants(ctx, courseID)
		if err != nil {
			return err
		}

		return s.outbox.WithTx(tx).Insert(ctx, model.EventCourseQuotaChanged, model.EnrollmentEvent{
			CourseID:     course.ID,
			CourseCode:   course.Code,
			CourseName:   course.Name,
			NewCapacity:  course.Capacity,
			Participants: participants,
		})
	})

	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas tidak ditemukan")
	case err != nil:
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota kelas berhasil diperbarui", course)
}
