package middleware

import (
	"io"

	"pemrograman-code/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// SetupGlobal memasang middleware global: requestid, logger JSON, cors.
func SetupGlobal(app *fiber.App, writer io.Writer) {
	app.Use(requestid.New())
	app.Use(config.NewFiberLogger(writer))
	app.Use(cors.New())
}
