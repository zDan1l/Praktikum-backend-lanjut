package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger membuat logger terstruktur yang menulis ke dua tujuan sekaligus:
// layar (stdout) dan file logs/app.log yang dirotasi otomatis.
func NewLogger() *slog.Logger {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic("gagal membuat folder logs: " + err.Error())
	}

	rotator := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "app.log"),
		MaxSize:    10, // rotasi setiap 10 MB
		MaxBackups: 5,
		MaxAge:     14, // hapus file >14 hari
		Compress:   true,
	}

	writer := io.MultiWriter(os.Stdout, rotator)
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: parseLevel(GetEnv("LOG_LEVEL", "info")),
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLogWriter untuk kompatibilitas lama (jika masih dipakai)
func GetLogWriter() io.Writer {
	_ = os.MkdirAll("logs", 0755)
	rotator := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "app.log"),
		MaxSize:    5,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   false,
	}
	return io.MultiWriter(os.Stdout, rotator)
}

// NewFiberLogger untuk kompatibilitas lama
func NewFiberLogger(writer io.Writer) *slog.Logger {
	return NewLogger()
}
