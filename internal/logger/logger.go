package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string) *slog.Logger {
	logLevel := parseLogLevel(level)

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler)
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "info", "":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "error":
		return slog.LevelError
	case "warn", "warning":
		return slog.LevelWarn
	default:
		return slog.LevelInfo

	}
}
