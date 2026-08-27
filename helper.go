package main

import (
	"context"
	"time"
	"pemrograman-code/app/model"
	"github.com/gofiber/fiber/v2"
)
// reqCtx memberi batas waktu untuk setiap operasi basis data.
// Tanpa batas waktu, satu query yang menggantung dapat menahan koneksi
// selamanya dan lama-lama menghabiskan seluruh isi pool.
func reqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
 return context.WithTimeout(c.UserContext(), 5*time.Second)
}
