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
	"pemrograman-code/helper"
	"pemrograman-code/route"
)

const minSecretLength = 32

func main() {
	config.LoadEnv()
	logger := config.NewLogger()

	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek", slog.Int("minimal_karakter", minSecretLength))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	// Repository
	studentRepo := repository.NewStudentRepository(pool)
	prestasiRepo := repository.NewPrestasiRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)

	// Service/Handler
	studentHandler := service.NewStudentHandler(studentRepo)
	prestasiHandler := service.NewPrestasiHandler(prestasiRepo)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtManager,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour)

	app := config.NewApp(logger, route.Dependencies{
		Pool:            pool,
		JWT:             jwtManager,
		StudentHandler:  studentHandler,
		PrestasiHandler: prestasiHandler,
		AuthService:     authService,
	})

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
