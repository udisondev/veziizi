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

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	adminRepo "github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/handlers"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/middleware"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/geoip"
	"github.com/udisondev/veziizi/backend/internal/pkg/httputil"
	"github.com/udisondev/veziizi/backend/internal/pkg/logging"
	"github.com/udisondev/veziizi/backend/internal/pkg/metrics"
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

	logFile, err := logging.Setup(cfg.App.LogLevel, cfg.App.LogFile)
	if err != nil {
		slog.Error("failed to setup logger", "error", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer func() {
			if err := logFile.Close(); err != nil {
				slog.Error("failed to close log file", "error", err)
			}
		}()
	}

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

	// Metrics server (Prometheus + pprof) на отдельном порту
	if cfg.Metrics.Enabled {
		metricsSrv := metrics.NewServer(cfg.Metrics.Addr)
		go func() {
			if err := metricsSrv.Start(); err != nil {
				slog.Error("metrics server error", slog.String("error", err.Error()))
			}
		}()
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
				slog.Error("failed to shutdown metrics server", slog.String("error", err.Error()))
			}
		}()
	}

	// Request ID + Recoverer + Metrics — на корневом роутере, общий для всех
	server.Router().Use(chiMiddleware.RequestID)
	server.Router().Use(chiMiddleware.Recoverer)
	server.Router().Use(middleware.Metrics)

	// Health endpoints — без auth/CSRF/rate limiter
	healthHandler := handlers.NewHealthHandler(f.MustPool())
	server.Router().Group(func(r chi.Router) {
		healthHandler.RegisterRoutes(r)
	})

	// API routes с полным middleware stack
	server.Router().Group(func(r chi.Router) {
		r.Use(middleware.SecurityHeaders(cfg)) // SEC-011
		r.Use(middleware.CORS(cfg))            // SEC-010
		r.Use(middleware.BodyLimit())          // SEC-015
		r.Use(middleware.RequireAuth(sessionManager))
		r.Use(middleware.CheckMemberStatus(sessionManager, f.MembersProjection()))
		r.Use(middleware.EventMetaEnricher(sessionManager)) // Добавляем metadata для аудита событий
		r.Use(middleware.RateLimiter(sessionManager, f.SessionAnalyzer()))
		r.Use(middleware.CSRFProtection()) // SEC-005

		orgHandler := handlers.NewOrganizationHandler(f.OrganizationService(), f.OrganizationRatingsProjection(), sessionManager)
		orgHandler.RegisterRoutes(r)

		authHandler := handlers.NewAuthHandler(f.MembersProjection(), f.FreightRequestsProjection(), f.OrganizationService(), sessionManager, f.SessionAnalyzer(), geoIPService)
		authHandler.RegisterRoutes(r)

		// Password reset handler (public routes for forgot/reset password)
		passwordResetHandler := handlers.NewPasswordResetHandler(
			f.MembersProjection(),
			f.PasswordResetProjection(),
			f.EmailTemplatesProjection(),
			f.EmailProvider(),
			cfg,
		)
		passwordResetHandler.RegisterRoutes(r)

		frHandler := handlers.NewFreightRequestHandler(f.FreightRequestService(), f.OrganizationService(), f.FreightRequestsProjection(), f.MembersProjection(), sessionManager)
		frHandler.RegisterRoutes(r)

		historyHandler := handlers.NewHistoryHandler(f.HistoryService(), f.FreightRequestService(), sessionManager)
		historyHandler.RegisterRoutes(r)

		geoHandler := handlers.NewGeoHandler(f.GeoProjection())
		geoHandler.RegisterRoutes(r)

		// Notification handler
		notificationHandler := handlers.NewNotificationHandler(
			f.NotificationService(),
			sessionManager,
			cfg,
		)
		notificationHandler.RegisterRoutes(r)
		if cfg.Telegram.BotUsername != "" {
			slog.Info("telegram notifications enabled", slog.String("bot", cfg.Telegram.BotUsername))
		}

		// Subscriptions handler (подписки на заявки)
		subscriptionsHandler := handlers.NewSubscriptionsHandler(
			f.FreightSubscriptionsProjection(),
			f.GeoProjection(),
			sessionManager,
		)
		subscriptionsHandler.RegisterRoutes(r)

		// Support handler (user tickets)
		supportHandler := handlers.NewSupportHandler(
			f.SupportService(),
			f.SupportTicketsProjection(),
			sessionManager,
		)
		supportHandler.RegisterRoutes(r)

		// Admin subrouter with RequireAdminAuth
		adminHandler := handlers.NewAdminHandler(f.AdminService(), adminRepository, adminSessionManager, f.ReviewService(), f.ReviewsProjection(), f.FraudDataProjection())
		r.Route("/api/v1/admin", func(r chi.Router) {
			r.Use(middleware.RequireAdminAuth(adminSessionManager))
			adminHandler.RegisterRoutes(r)

			// Admin support handler
			adminSupportHandler := handlers.NewAdminSupportHandler(
				f.SupportService(),
				f.SupportTicketsProjection(),
				adminSessionManager,
			)
			adminSupportHandler.RegisterRoutes(r)

			// Admin email templates handler
			adminEmailTemplatesHandler := handlers.NewAdminEmailTemplatesHandler(
				f.EmailTemplatesProjection(),
				adminSessionManager,
			)
			adminEmailTemplatesHandler.RegisterRoutes(r)
		})

		// Dev handler (only in development mode)
		// SEC-001: Двойная защита - проверка IsDevelopment() + DevOnly middleware
		if cfg.IsDevelopment() {
			r.Route("/api/v1/dev", func(r chi.Router) {
				r.Use(middleware.DevOnly(cfg))
				devHandler := handlers.NewDevHandler(cfg, f.MembersProjection(), f.OrganizationService(), sessionManager)
				devHandler.RegisterRoutesWithRouter(r)
			})
			slog.Info("dev user switcher enabled (development mode only)")
		}
	})

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

