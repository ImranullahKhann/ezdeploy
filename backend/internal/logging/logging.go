package logging

import (
	"log/slog"
	"os"
)

func New(level string) *slog.Logger {
	var handlerOptions slog.HandlerOptions
	if parsedLevel, err := parseLevel(level); err == nil {
		handlerOptions.Level = parsedLevel
	}

	handler := slog.NewJSONHandler(os.Stdout, &handlerOptions)
	return slog.New(handler)
}

func parseLevel(level string) (slog.Level, error) {
	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(level)); err != nil {
		return 0, err
	}
	return parsed, nil
}
