package helper

import (
	"pemrograman-code/app/model"

	"github.com/gofiber/fiber/v2"
)

// === Cara pakai di handler ===
// helper.Success(c, 200, "pesan", data)
// helper.SuccessList(c, "pesan", data, meta) // untuk list + pagination
// helper.Created(c, "pesan", data, "/api/v1/.../id") // 201 + header Location
// helper.Fail(c, 404, "tidak ditemukan")
// helper.FailValidation(c, map[string]string{"field":"pesan"})
// helper.NoContent(c) // 204

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func SuccessList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: false, Message: message,
	})
}

func FailValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: false, Message: "validasi gagal", Errors: errs,
	})
}

// Alias kompatibilitas (handler lama pakai Ok/OkList, tetap jalan)
// Prefer pakai Success/SuccessList untuk kode baru
func Ok(c *fiber.Ctx, message string, data any) error {
	return Success(c, fiber.StatusOK, message, data)
}
func OkList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return SuccessList(c, message, data, meta)
}
