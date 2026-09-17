package config

import (
	"log"
	"log/slog"
	"os"
	"strconv"

	"pemrograman-code/helper"
	"pemrograman-code/middleware"
	"pemrograman-code/route"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

// ===== environment =====

// LoadEnv memuat file .env. Bila tidak ada, pakai environment sistem.
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("file .env tidak ditemukan, memakai environment sistem")
	}
}

func GetEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func GetEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// ===== logger =====

// NewLogger logger terstruktur JSON ke stdout.
func NewLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	return logger
}

// ===== aplikasi Fiber =====

func NewApp(logger *slog.Logger, deps route.Dependencies) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "Praktikum Backend Lanjut"),
		ErrorHandler: errorHandler(logger),
		BodyLimit:    1 * 1024 * 1024, // batas body 1 MB
	})

	middleware.Register(app, logger, GetEnv("ALLOWED_ORIGINS", ""))
	route.Setup(app, deps)

	// semua route yang tidak terdaftar jatuh ke sini
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}

func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		message := "terjadi error pada server"
		if e, ok := err.(*fiber.Error); ok {
			status, message = e.Code, e.Message
		}
		logger.Error("unhandled_error",
			slog.String("path", c.Path()),
			slog.String("error", err.Error()),
		)
		return helper.Fail(c, status, message)
	}
}
