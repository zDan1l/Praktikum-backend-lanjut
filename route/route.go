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

// Setup cuma mapping URL -> handler, tanpa if/validasi.
// Cara tambah resource baru (contoh: buku):
// 1. Buat model, repository, handler di app/
// 2. Tambah 2 baris di main.go: bukuRepo := repository.NewBukuRepository(pool); bukuHandler := service.NewBukuHandler(bukuRepo)
// 3. Tambah 1 baris di Setup: registerBukuRoutes(api, bukuHandler)
// 4. Tulis func registerBukuRoutes di bawah (copy dari student/prestasi)
func Setup(app *fiber.App, pool *pgxpool.Pool, studentHandler *service.StudentHandler, prestasiHandler *service.PrestasiHandler) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	api := app.Group("/api/v1")
	registerHealth(api, pool)
	registerStudentRoutes(api, studentHandler)
	registerPrestasiRoutes(api, prestasiHandler)
}

func registerHealth(api fiber.Router, pool *pgxpool.Pool) {
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", fiber.Map{"timestamp": time.Now()})
	})
}

func registerStudentRoutes(api fiber.Router, h *service.StudentHandler) {
	g := api.Group("/students", middleware.RequireJSON)
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Post("/", h.Create)
	g.Put("/:id", h.Replace)
	g.Patch("/:id", h.Patch)
	g.Delete("/:id", h.Delete)
}

func registerPrestasiRoutes(api fiber.Router, h *service.PrestasiHandler) {
	g := api.Group("/prestasi", middleware.RequireJSON)
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Post("/", h.Create)
	g.Put("/:id", h.Replace)
	g.Patch("/:id", h.Patch)
	g.Delete("/:id", h.Delete)
}
