package logging

import (
	"io"
	"log/slog"
	"os"
)

// Setup configures slog with JSON handler writing to stdout and optionally to a file.
// Returns the log file (nil if no file) which the caller should close via defer.
func Setup(levelStr, logFilePath string) (*os.File, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}

	var w io.Writer = os.Stdout

	var file *os.File
	if logFilePath != "" {
		var err error
		file, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		w = io.MultiWriter(os.Stdout, file)
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	return file, nil
}
