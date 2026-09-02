package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"pemrograman-code/app/repository"
	"pemrograman-code/config"
	"pemrograman-code/database"
)

func main() {
	// 1. Konfigurasi
	config.LoadEnv()

	// 2. Koneksi basis data
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// 3. Perakitan: pool -> repository -> handler
	studentRepository := repository.NewStudentRepository(pool)
	studentHandler := NewStudentHandler(studentRepository)

	// 4. Aplikasi Fiber
	app := fiber.New(fiber.Config{
		AppName: "Praktikum Backend Lanjut - Modul 3",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			pesan := "terjadi kesalahan pada server"
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				pesan = e.Message
			}
			return fail(c, status, pesan)
		},
	})

	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${locals:requestid} ${method} ${path} ${status} ${latency}\n",
	}))
	app.Use(cors.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return ok(c, "server dan database berjalan", fiber.Map{"timestamp": time.Now()})
	})

	s := api.Group("/students", requireJSON)
	s.Get("/", studentHandler.List)
	s.Get("/:id", studentHandler.Get)
	s.Post("/", studentHandler.Create)
	s.Put("/:id", studentHandler.Replace)
	s.Patch("/:id", studentHandler.Patch)
	s.Delete("/:id", studentHandler.Delete)

	// Endpoint tidak dikenal
	app.Use(func(c *fiber.Ctx) error {
		return fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	port := config.GetEnv("APP_PORT", "3000")
	fmt.Println("Server berjalan di http://localhost:" + port)
	log.Fatal(app.Listen(":" + port))
}
