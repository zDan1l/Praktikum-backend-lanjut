// Command redeliver mengirim ulang (enqueue) beberapa SALINAN task notifikasi
// untuk satu baris outbox, dipakai menguji idempotency (Modul 11, C.5).
//
//	go run ./tools/redeliver -outbox <id> -copies 5
//
// Setiap salinan memakai TaskID yang berbeda sehingga lolos dari proteksi
// anti-duplikat Asynq (TaskID) — tujuannya memang membuat banyak salinan
// sampai ke worker. Worker yang idempoten harus tetap menghasilkan tepat satu
// notifikasi dan satu email, berapa pun jumlah salinannya.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hibiken/asynq"

	"pemrograman-code/app/model"
	"pemrograman-code/config"
	"pemrograman-code/database"
)

func main() {
	outboxID := flag.Int64("outbox", 0, "id baris outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 5, "jumlah salinan task yang dibuat")
	flag.Parse()

	if *outboxID <= 0 || *copies <= 0 {
		fmt.Fprintln(os.Stderr, "pemakaian: go run ./tools/redeliver -outbox <id> -copies <n>")
		os.Exit(1)
	}

	config.LoadEnv()

	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal terhubung ke database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Ambil isi event dari outbox. published_at diabaikan: tool ini sengaja
	// memakai baris itu apa pun statusnya.
	var eventType string
	var payload []byte
	err = pool.QueryRow(ctx,
		`SELECT event_type, payload FROM outbox WHERE id = $1`, *outboxID,
	).Scan(&eventType, &payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "outbox %d tidak ditemukan: %v\n", *outboxID, err)
		os.Exit(1)
	}

	job := model.NotificationJob{
		OutboxID:  *outboxID,
		EventType: eventType,
		Payload:   payload,
	}
	data, err := json.Marshal(job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal menyusun payload: %v\n", err)
		os.Exit(1)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	for i := 1; i <= *copies; i++ {
		task := asynq.NewTask(model.TaskSendNotification, data,
			// TaskID berbeda per salinan supaya tidak saling menolak.
			asynq.TaskID(fmt.Sprintf("redeliver-%d-%d", *outboxID, i)),
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
			asynq.Retention(24*time.Hour),
		)
		info, err := client.EnqueueContext(ctx, task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gagal enqueue salinan %d: %v\n", i, err)
			os.Exit(1)
		}
		fmt.Printf("salinan %d di-enqueue: id=%s queue=%s\n", i, info.ID, info.Queue)
	}
	fmt.Printf("selesai: %d salinan task untuk outbox %d dikirim ke Redis\n", *copies, *outboxID)
}
