package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

var Logger *slog.Logger

func init() {
	Init("info")
}

func Init(level string) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	w := os.Stdout
	Logger = slog.New(tint.NewHandler(w, &tint.Options{
		Level:      logLevel,
		TimeFormat: time.Kitchen,
	}))
}
