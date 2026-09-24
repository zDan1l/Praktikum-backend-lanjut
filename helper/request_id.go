package helper

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// RequestID mengambil request_id dari Locals, buat baru bila belum ada.
// Dipakai ErrorHandler dan RequestLogger agar dua baris log terikat satu id.
func RequestID(c *fiber.Ctx) string {
	if v, ok := c.Locals("requestid").(string); ok && v != "" {
		return v
	}
	id := uuid.NewString()
	c.Locals("requestid", id)
	return id
}
