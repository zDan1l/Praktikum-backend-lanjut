package route

import (
	"context"
	"time"

	"pemrograman-code/app/service"
	"pemrograman-code/helper"
	"pemrograman-code/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Setup mendaftarkan semua route tanpa business rules / validasi if apapun.
// Hanya pemetaan URL -> handler.
func Setup(app *fiber.App, pool *pgxpool.Pool, h *service.StudentHandler) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return helper.Ok(c, "server dan database berjalan", fiber.Map{"timestamp": time.Now()})
	})

	s := api.Group("/students", middleware.RequireJSON)
	s.Get("/", h.List)
	s.Get("/:id", h.Get)
	s.Post("/", h.Create)
	s.Put("/:id", h.Replace)
	s.Patch("/:id", h.Patch)
	s.Delete("/:id", h.Delete)

	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})
}
