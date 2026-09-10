package config

import (
	"log/slog"

	"pemrograman-code/app/service"
	"pemrograman-code/helper"
	"pemrograman-code/middleware"
	"pemrograman-code/route"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AppConfig struct {
	Port string
}

func LoadAppConfig() AppConfig {
	LoadEnv()
	return AppConfig{
		Port: GetEnv("APP_PORT", "3000"),
	}
}

// NewApp merakit aplikasi: membuat Fiber, memasang middleware, mendaftarkan route.
func NewApp(logger *slog.Logger, pool *pgxpool.Pool, studentService *service.StudentHandler, prestasiService *service.PrestasiHandler) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "Praktikum Backend Lanjut"),
		ErrorHandler: newErrorHandler(logger),
	})

	middleware.Register(app, logger)
	route.Setup(app, pool, studentService, prestasiService)

	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}

func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		message := "terjadi error pada server"
		if e, ok := err.(*fiber.Error); ok {
			status = e.Code
			message = e.Message
		}
		logger.Error("unhandled_error",
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.String("error", err.Error()),
		)
		return helper.Fail(c, status, message)
	}
}
