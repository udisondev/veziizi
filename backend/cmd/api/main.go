package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events" // register events
	_ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"   // register events
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"   // register events
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"         // register events
	_ "github.com/udisondev/veziizi/backend/internal/domain/support/events"        // register events

	adminRepo "github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/handlers"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/middleware"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/geoip"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// SEC-004: Configure trusted proxies for IP extraction
	if cfg.HTTP.TrustedProxies != "" {
		proxies := strings.Split(cfg.HTTP.TrustedProxies, ",")
		httputil.SetTrustedProxies(proxies)
		slog.Info("trusted proxies configured", slog.Int("count", len(proxies)))
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create factory IoC container - all dependencies are lazily initialized
	f := factory.New(cfg)
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
	}()

	// Create GeoIP service (optional, works without database file)
	geoIPService := geoip.NewService(cfg.GeoIP.DatabasePath)
	defer func() {
		if err := geoIPService.Close(); err != nil {
			slog.Error("failed to close geoip service", slog.String("error", err.Error()))
		}
	}()

	// Инициализация rate limiter'а с конфигом (до middleware)
	middleware.InitRateLimiter(&cfg.RateLimit)

	sessionManager := session.NewManager(cfg)
	adminSessionManager := session.NewAdminManager(cfg)

	// Repositories (not managed by factory)
	adminRepository := adminRepo.NewRepository(f.DB())

	// HTTP server and handlers
	server := http.NewServer(cfg)

	// Apply middleware to all routes
	server.Router().Use(middleware.SecurityHeaders(cfg)) // SEC-011
	server.Router().Use(middleware.CORS(cfg))            // SEC-010
	server.Router().Use(middleware.BodyLimit())          // SEC-015
	server.Router().Use(middleware.RequireAuth(sessionManager))
	server.Router().Use(middleware.CheckMemberStatus(sessionManager, f.MembersProjection()))
	server.Router().Use(middleware.EventMetaEnricher(sessionManager)) // Добавляем metadata для аудита событий
	server.Router().Use(middleware.RateLimiter(sessionManager, f.SessionAnalyzer()))
	server.Router().Use(middleware.CSRFProtection()) // SEC-005

	orgHandler := handlers.NewOrganizationHandler(f.OrganizationService(), f.OrganizationRatingsProjection(), sessionManager)
	orgHandler.RegisterRoutes(server.Router())

	authHandler := handlers.NewAuthHandler(f.MembersProjection(), f.FreightRequestsProjection(), f.OrganizationService(), sessionManager, f.SessionAnalyzer(), geoIPService)
	authHandler.RegisterRoutes(server.Router())

	// Password reset handler (public routes for forgot/reset password)
	passwordResetHandler := handlers.NewPasswordResetHandler(
		f.MembersProjection(),
		f.PasswordResetProjection(),
		f.EmailTemplatesProjection(),
		f.EmailProvider(),
		cfg,
	)
	passwordResetHandler.RegisterRoutes(server.Router())

	adminHandler := handlers.NewAdminHandler(f.AdminService(), adminRepository, adminSessionManager, f.ReviewService(), f.ReviewsProjection(), f.FraudDataProjection())
	// Register login on main router (no admin auth required)
	server.Router().HandleFunc("/api/v1/admin/auth/login", adminHandler.Login).Methods("POST")
	// All other admin routes require admin auth
	adminRouter := server.Router().PathPrefix("/api/v1/admin").Subrouter()
	adminRouter.Use(middleware.RequireAdminAuth(adminSessionManager))
	adminHandler.RegisterRoutes(adminRouter)

	frHandler := handlers.NewFreightRequestHandler(f.FreightRequestService(), f.OrganizationService(), f.FreightRequestsProjection(), f.MembersProjection(), sessionManager)
	frHandler.RegisterRoutes(server.Router())

	historyHandler := handlers.NewHistoryHandler(f.HistoryService(), f.FreightRequestService(), sessionManager)
	historyHandler.RegisterRoutes(server.Router())

	geoHandler := handlers.NewGeoHandler(f.GeoProjection())
	geoHandler.RegisterRoutes(server.Router())

	// Notification handler
	notificationHandler := handlers.NewNotificationHandler(
		f.NotificationService(),
		sessionManager,
		cfg,
	)
	notificationHandler.RegisterRoutes(server.Router())
	if cfg.Telegram.BotUsername != "" {
		slog.Info("telegram notifications enabled", slog.String("bot", cfg.Telegram.BotUsername))
	}

	// Subscriptions handler (подписки на заявки)
	subscriptionsHandler := handlers.NewSubscriptionsHandler(
		f.FreightSubscriptionsProjection(),
		f.GeoProjection(),
		sessionManager,
	)
	subscriptionsHandler.RegisterRoutes(server.Router())

	// Support handler (user tickets)
	supportHandler := handlers.NewSupportHandler(
		f.SupportService(),
		f.SupportTicketsProjection(),
		sessionManager,
	)
	supportHandler.RegisterRoutes(server.Router())

	// Admin support handler (admin tickets management)
	adminSupportHandler := handlers.NewAdminSupportHandler(
		f.SupportService(),
		f.SupportTicketsProjection(),
		adminSessionManager,
	)
	adminSupportHandler.RegisterRoutes(adminRouter)

	// Admin email templates handler
	adminEmailTemplatesHandler := handlers.NewAdminEmailTemplatesHandler(
		f.EmailTemplatesProjection(),
		adminSessionManager,
	)
	adminEmailTemplatesHandler.RegisterRoutes(adminRouter)

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
			stop()
		}
	}()

	<-ctx.Done()

	slog.Info("shutting down...")

	// Stop rate limiter cleanup goroutine
	middleware.StopRateLimiterCleanup()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
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
