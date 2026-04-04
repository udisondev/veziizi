package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
)

type Server struct {
	router chi.Router
	srv    *http.Server
}

func NewServer(cfg *config.Config) *Server {
	router := chi.NewRouter()

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	return &Server{
		router: router,
		srv:    srv,
	}
}

func (s *Server) Router() chi.Router {
	return s.router
}

func (s *Server) Start() error {
	slog.Info("starting HTTP server", slog.String("addr", s.srv.Addr))

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("shutting down HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.srv.Shutdown(shutdownCtx)
}
