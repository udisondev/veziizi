package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// contextHandler оборачивает slog.Handler и автоматически добавляет request_id из контекста.
type contextHandler struct {
	inner slog.Handler
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := chiMiddleware.GetReqID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}

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

	jsonHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(&contextHandler{inner: jsonHandler}))

	return file, nil
}
