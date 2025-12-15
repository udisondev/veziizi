package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	adminApp "codeberg.org/udison/veziizi/backend/internal/application/admin"
	orgApp "codeberg.org/udison/veziizi/backend/internal/application/organization"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events" // register events
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	adminRepo "codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/admin"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/handlers"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logFile, err := setupLogger(cfg.App.LogLevel)
	if err != nil {
		slog.Error("failed to setup logger", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer logFile.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("connected to database")

	txManager := dbtx.NewTxExecutor(pool)

	eventStore := eventstore.NewPostgresStore(txManager)

	wmLogger := watermill.NewSlogLogger(slog.Default())
	publisher, err := messaging.NewEventPublisher(pool, wmLogger)
	if err != nil {
		slog.Error("failed to create event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer publisher.Close()

	sessionManager := session.NewManager(cfg)
	adminSessionManager := session.NewAdminManager(cfg)

	// Projections (read-only)
	membersProjection := projections.NewMembersProjection(txManager)
	invitationsProjection := projections.NewInvitationsProjection(txManager)
	pendingOrgsProjection := projections.NewPendingOrganizationsProjection(txManager)

	// Repositories
	adminRepository := adminRepo.NewRepository(txManager)

	// Application services
	orgService := orgApp.NewService(txManager, eventStore, publisher, invitationsProjection)
	adminService := adminApp.NewService(txManager, eventStore, publisher, pendingOrgsProjection)

	// HTTP server and handlers
	server := http.NewServer(cfg)

	orgHandler := handlers.NewOrganizationHandler(orgService, sessionManager)
	orgHandler.RegisterRoutes(server.Router())

	authHandler := handlers.NewAuthHandler(membersProjection, sessionManager)
	authHandler.RegisterRoutes(server.Router())

	adminHandler := handlers.NewAdminHandler(adminService, adminRepository, adminSessionManager)
	adminHandler.RegisterRoutes(server.Router())

	go func() {
		if err := server.Start(); err != nil {
			slog.Error("server error", slog.String("error", err.Error()))
			cancel()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown server", slog.String("error", err.Error()))
	}
}

func setupLogger(levelStr string) (*os.File, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		level = slog.LevelInfo
	}

	file, err := os.OpenFile("current.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	return file, nil
}
