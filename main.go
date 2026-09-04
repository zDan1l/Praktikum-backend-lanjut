package main

import (
	"context"
	"fmt"
	"log"

	"pemrograman-code/app/repository"
	"pemrograman-code/app/service"
	"pemrograman-code/config"
	"pemrograman-code/database"
	"pemrograman-code/helper"
	"pemrograman-code/middleware"
	"pemrograman-code/route"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.LoadAppConfig()
	writer := config.GetLogWriter()

	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	studentRepo := repository.NewStudentRepository(pool)
	studentHandler := service.NewStudentHandler(studentRepo)

	app := fiber.New(fiber.Config{
		AppName: "Praktikum Backend Lanjut - Modul 4",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			pesan := "terjadi kesalahan pada server"
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				pesan = e.Message
			}
			return helper.Fail(c, status, pesan)
		},
	})

	middleware.SetupGlobal(app, writer)
	route.Setup(app, pool, studentHandler)

	fmt.Println("Server berjalan di http://localhost:" + cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
