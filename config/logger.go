package config

import (
	"io"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"gopkg.in/natefinch/lumberjack.v2"
)

// GetLogWriter mengembalikan writer yang menulis ke layar sekaligus ke logs/app.log dengan rotasi.
// Rotasi: max 5MB per file, simpan 3 backup, hapus setelah 28 hari.
func GetLogWriter() io.Writer {
	_ = os.MkdirAll("logs", 0755)
	fileWriter := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    5, // MB
		MaxBackups: 3,
		MaxAge:     28, // hari
		Compress:   false,
	}
	return io.MultiWriter(os.Stdout, fileWriter)
}

// NewFiberLogger membuat middleware logger yang mencatat 1 baris JSON per request.
// Field wajib: request_id, method, path, status, duration.
func NewFiberLogger(writer io.Writer) fiber.Handler {
	return logger.New(logger.Config{
		Output:     writer,
		Format:     "{\"request_id\":\"${locals:requestid}\",\"method\":\"${method}\",\"path\":\"${path}\",\"status\":${status},\"duration\":\"${latency}\"}\n",
		TimeFormat: "2006-01-02T15:04:05Z07:00",
	})
}
