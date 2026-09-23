package main

import (
	"context"
	"log/slog"
	"os"
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

	studentHandler := service.NewStudentHandler(repository.NewStudentRepository(pool))
	userRepo := repository.NewUserRepository(pool)
	authService := service.NewAuthService(
		userRepo,
		repository.NewTokenRepository(pool),
		jwtManager,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
	)

	app := config.NewApp(logger, route.Dependencies{
		Pool:           pool,
		JWT:            jwtManager,
		Users:          userRepo,
		StudentHandler: studentHandler,
		AuthService:    authService,
	})

	port := config.GetEnv("APP_PORT", "3000")
	logger.Info("server berjalan", slog.String("port", port))
	if err := app.Listen(":" + port); err != nil {
		logger.Error("server berhenti", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
