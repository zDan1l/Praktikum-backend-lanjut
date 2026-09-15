package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pemrograman-code/app/repository"
	"pemrograman-code/app/service"
	"pemrograman-code/config"
	"pemrograman-code/database"
)

func main() {
	// 1. Config & logger
	config.LoadEnv()
	logger := config.NewLogger()

	// 2. Database pool
	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// 3. Repository (satu per tabel)
	// Cara tambah tabel baru: copy 1 baris di bawah, ganti nama
	studentRepo := repository.NewStudentRepository(pool)
	prestasiRepo := repository.NewPrestasiRepository(pool)

	// 4. Handler (satu per repository)
	studentHandler := service.NewStudentHandler(studentRepo)
	prestasiHandler := service.NewPrestasiHandler(prestasiRepo)

	// 5. Rakit aplikasi (route terdaftar di route/route.go)
	app := config.NewApp(logger, pool, studentHandler, prestasiHandler)

	port := config.GetEnv("APP_PORT", "3000")

	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("server berjalan", slog.String("port", port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("sinyal berhenti diterima, menutup server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("gagal menutup server dengan rapi", slog.String("error", err.Error()))
	}

	logger.Info("server berhenti dengan rapi")
}
