package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events" // register events
	_ "codeberg.org/udison/veziizi/backend/internal/domain/order/events"          // register events
	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"   // register events
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/pkg/geoip"
	adminRepo "codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/admin"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/filestorage"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/handlers"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/middleware"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
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
	defer func() {
		if err := logFile.Close(); err != nil {
			slog.Error("failed to close log file", slog.String("error", err.Error()))
		}
	}()

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
	fileStorage := filestorage.NewPostgresStorage(txManager)

	wmLogger := watermill.NewSlogLogger(slog.Default())
	publisher, err := messaging.NewEventPublisher(pool, wmLogger)
	if err != nil {
		slog.Error("failed to create event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			slog.Error("failed to close publisher", slog.String("error", err.Error()))
		}
	}()

	// Create factory
	f := factory.New(txManager, eventStore, publisher, fileStorage)

	// Create GeoIP service (optional, works without database file)
	geoIPService := geoip.NewService(cfg.GeoIP.DatabasePath)
	defer func() {
		if err := geoIPService.Close(); err != nil {
			slog.Error("failed to close geoip service", slog.String("error", err.Error()))
		}
	}()

	sessionManager := session.NewManager(cfg)
	adminSessionManager := session.NewAdminManager(cfg)

	// Repositories (not managed by factory)
	adminRepository := adminRepo.NewRepository(txManager)

	// HTTP server and handlers
	server := http.NewServer(cfg)

	// Apply middleware to all routes
	server.Router().Use(middleware.SecurityHeaders(cfg)) // SEC-011
	server.Router().Use(middleware.CORS(cfg))            // SEC-010
	server.Router().Use(middleware.BodyLimit())          // SEC-015
	server.Router().Use(middleware.RequireAuth(sessionManager))
	server.Router().Use(middleware.RateLimiter(sessionManager, f.SessionAnalyzer()))
	server.Router().Use(middleware.CSRFProtection()) // SEC-005

	orgHandler := handlers.NewOrganizationHandler(f.OrganizationService(), f.OrganizationRatingsProjection(), sessionManager)
	orgHandler.RegisterRoutes(server.Router())

	authHandler := handlers.NewAuthHandler(f.MembersProjection(), f.OrdersProjection(), f.OrganizationService(), sessionManager, f.SessionAnalyzer(), geoIPService)
	authHandler.RegisterRoutes(server.Router())

	adminHandler := handlers.NewAdminHandler(f.AdminService(), adminRepository, adminSessionManager, f.ReviewService(), f.ReviewsProjection(), f.FraudDataProjection())
	adminHandler.RegisterRoutes(server.Router())

	frHandler := handlers.NewFreightRequestHandler(f.FreightRequestService(), f.OrganizationService(), f.FreightRequestsProjection(), f.MembersProjection(), sessionManager)
	frHandler.RegisterRoutes(server.Router())

	orderHandler := handlers.NewOrderHandler(f.OrderService(), f.OrganizationService(), f.MembersProjection(), f.OrdersProjection(), sessionManager)
	orderHandler.RegisterRoutes(server.Router())

	historyHandler := handlers.NewHistoryHandler(f.HistoryService(), f.FreightRequestService(), f.OrderService(), sessionManager)
	historyHandler.RegisterRoutes(server.Router())

	// Dev handler (only in development mode)
	// SEC-001: Двойная защита - проверка IsDevelopment() + DevOnly middleware
	if cfg.IsDevelopment() {
		devRouter := server.Router().PathPrefix("/api/v1/dev").Subrouter()
		devRouter.Use(middleware.DevOnly(cfg))
		devHandler := handlers.NewDevHandler(cfg, f.MembersProjection(), f.OrganizationService(), sessionManager)
		devHandler.RegisterRoutesWithRouter(devRouter)
		slog.Info("dev user switcher enabled (development mode only)")
	}

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
