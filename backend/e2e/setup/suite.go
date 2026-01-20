// Package setup provides E2E test infrastructure for the veziizi API.
// It handles server startup, database management, and test lifecycle.
package setup

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	_ "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/notification/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/organization/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/review/events"
	_ "codeberg.org/udison/veziizi/backend/internal/domain/support/events"

	eventHandlers "codeberg.org/udison/veziizi/backend/internal/infrastructure/handlers"
	adminRepo "codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/admin"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	httpServer "codeberg.org/udison/veziizi/backend/internal/interfaces/http"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/handlers"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/middleware"
	"codeberg.org/udison/veziizi/backend/internal/interfaces/http/session"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/factory"
	"codeberg.org/udison/veziizi/backend/internal/pkg/geoip"
	"github.com/ThreeDotsLabs/watermill"
	wmSql "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Suite represents a test suite with shared infrastructure.
// Use NewSuite() to create a new suite for each test group.
type Suite struct {
	T       *testing.T
	BaseURL string
	Factory *factory.Factory
	Config  *config.Config

	server            *httpServer.Server
	listener          net.Listener
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	postgresContainer *PostgresContainer
	eventRouter       *message.Router
}

// SharedSuite is a singleton suite for tests that can share infrastructure.
// Use for read-only tests or tests with proper isolation via unique IDs.
var (
	sharedSuite     *Suite
	sharedSuiteOnce sync.Once
	sharedSuiteMu   sync.Mutex
)

// GetSharedSuite returns or creates a shared test suite.
// The shared suite is faster because it reuses the server and database connection.
// Use this for tests that don't need complete isolation.
func GetSharedSuite(t *testing.T) *Suite {
	sharedSuiteMu.Lock()
	defer sharedSuiteMu.Unlock()

	sharedSuiteOnce.Do(func() {
		suite, err := newSuite(t)
		if err != nil {
			t.Fatalf("failed to create shared suite: %v", err)
		}
		sharedSuite = suite

		// Cleanup will be handled by TestMain
	})

	// Update T reference for current test
	return &Suite{
		T:                 t,
		BaseURL:           sharedSuite.BaseURL,
		Factory:           sharedSuite.Factory,
		Config:            sharedSuite.Config,
		server:            sharedSuite.server,
		ctx:               sharedSuite.ctx,
		cancel:            sharedSuite.cancel,
		postgresContainer: sharedSuite.postgresContainer,
		eventRouter:       sharedSuite.eventRouter,
	}
}

// NewSuite creates a new isolated test suite.
// Use this for tests that need complete isolation.
func NewSuite(t *testing.T) *Suite {
	suite, err := newSuite(t)
	if err != nil {
		t.Fatalf("failed to create suite: %v", err)
	}

	t.Cleanup(func() {
		suite.Shutdown()
	})

	return suite
}

func newSuite(t *testing.T) (*Suite, error) {
	// Increase rate limits for tests (10000 requests per window)
	middleware.SetRateLimits(10000, 10000)

	// Increase session fraud rate limits for tests
	projections.SetSessionFraudLimits(100000, 100000)

	// Increase registration velocity limits for tests
	projections.RegistrationVelocity.MaxRegistrationsPerIPPerHour = 10000
	projections.RegistrationVelocity.MaxRegistrationsPerFingerprintPer24h = 10000

	// Increase password reset rate limits for tests
	projections.SetPasswordResetRateLimits(10000, 10000)

	// Disable logging in tests (or set to minimal level)
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	})))

	ctx, cancel := context.WithCancel(context.Background())

	// Start PostgreSQL container
	pgContainer, err := StartPostgres(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	cfg := testConfigWithDSN(pgContainer.DSN)

	f := factory.New(cfg)

	// Run migrations
	if err := runMigrations(cfg); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Seed geo data (countries, cities)
	if err := SeedGeoData(cfg); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to seed geo data: %w", err)
	}

	// Create test admin
	if err := CreateTestAdmin(cfg); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create test admin: %w", err)
	}

	// Initialize Watermill schema (explicit, like in chord)
	if err := initWatermillSchema(cfg); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to init watermill schema: %w", err)
	}

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Update config with actual address
	cfg.HTTP.Addr = fmt.Sprintf("127.0.0.1:%d", port)

	suite := &Suite{
		T:                 t,
		BaseURL:           baseURL,
		Factory:           f,
		Config:            cfg,
		listener:          listener,
		ctx:               ctx,
		cancel:            cancel,
		postgresContainer: pgContainer,
	}

	// Start event handlers (watermill subscribers for projections)
	if err := suite.startEventHandlers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start event handlers: %w", err)
	}

	// Start server
	suite.startServer()

	// Wait for server to be ready
	if err := suite.waitForServer(); err != nil {
		suite.Shutdown()
		return nil, fmt.Errorf("server failed to start: %w", err)
	}

	return suite, nil
}

// startEventHandlers sets up watermill subscribers to process events into lookup tables.
// This is essential for E2E tests because login requires members_lookup to be populated.
// Each handler needs its own subscriber with unique consumer group to receive all messages.
func (s *Suite) startEventHandlers() error {
	wmLogger := watermill.NewSlogLogger(slog.Default())
	pool := s.Factory.MustPool()
	db := s.Factory.DB()

	// Helper to create subscriber with unique consumer group
	createSubscriber := func(consumerGroup, topic string) (message.Subscriber, error) {
		sub, err := wmSql.NewSubscriber(
			wmSql.BeginnerFromPgx(pool),
			wmSql.SubscriberConfig{
				SchemaAdapter:  wmSql.DefaultPostgreSQLSchema{},
				OffsetsAdapter: wmSql.DefaultPostgreSQLOffsetsAdapter{},
				ConsumerGroup:  consumerGroup,
			},
			wmLogger,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create subscriber %s: %w", consumerGroup, err)
		}
		if err := sub.SubscribeInitialize(topic); err != nil {
			return nil, fmt.Errorf("failed to initialize subscriber %s: %w", consumerGroup, err)
		}
		return sub, nil
	}

	// Create router
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return fmt.Errorf("failed to create router: %w", err)
	}

	// Organization event handlers - each needs its own subscriber
	membersSub, err := createSubscriber("e2e_members", "organization.events")
	if err != nil {
		return err
	}
	membersHandler := eventHandlers.NewMembersHandler(db)
	router.AddNoPublisherHandler("members", "organization.events", membersSub, membersHandler.Handle)

	orgsSub, err := createSubscriber("e2e_organizations", "organization.events")
	if err != nil {
		return err
	}
	organizationsHandler := eventHandlers.NewOrganizationsHandler(s.Factory.OrganizationsProjection(), s.Factory.FreightRequestsProjection())
	router.AddNoPublisherHandler("organizations", "organization.events", orgsSub, organizationsHandler.Handle)

	invSub, err := createSubscriber("e2e_invitations", "organization.events")
	if err != nil {
		return err
	}
	invitationsHandler := eventHandlers.NewInvitationsHandler(db)
	router.AddNoPublisherHandler("invitations", "organization.events", invSub, invitationsHandler.Handle)

	pendingSub, err := createSubscriber("e2e_pending_orgs", "organization.events")
	if err != nil {
		return err
	}
	pendingOrgsHandler := eventHandlers.NewPendingOrganizationsHandler(db)
	router.AddNoPublisherHandler("pending_orgs", "organization.events", pendingSub, pendingOrgsHandler.Handle)

	// Freight request event handlers
	frSub, err := createSubscriber("e2e_freight_requests", "freightrequest.events")
	if err != nil {
		return err
	}
	freightRequestsHandler := eventHandlers.NewFreightRequestsHandler(db, s.Factory.EventStore())
	router.AddNoPublisherHandler("freight_requests", "freightrequest.events", frSub, freightRequestsHandler.Handle)

	// Support event handlers
	supportSub, err := createSubscriber("e2e_support_tickets", "support.events")
	if err != nil {
		return err
	}
	supportTicketsHandler := eventHandlers.NewSupportTicketsHandler(db)
	router.AddNoPublisherHandler("support_tickets", "support.events", supportSub, supportTicketsHandler.Handle)

	// Fraudster handler (for marking organizations as fraudsters)
	fraudsterSub, err := createSubscriber("e2e_fraudster", "organization.events")
	if err != nil {
		return err
	}
	fraudsterHandler := eventHandlers.NewFraudsterHandler(s.Factory.ReviewService(), s.Factory.ReviewsProjection(), s.Factory.FraudDataProjection())
	router.AddNoPublisherHandler("fraudster", "organization.events", fraudsterSub, fraudsterHandler.Handle)

	s.eventRouter = router

	// Start router in background
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := router.Run(s.ctx); err != nil {
			slog.Error("event router error", slog.String("error", err.Error()))
		}
	}()

	// Wait for router to be running
	<-router.Running()

	return nil
}

func (s *Suite) startServer() {
	geoIPService := geoip.NewService("")
	sessionManager := session.NewManager(s.Config)
	adminSessionManager := session.NewAdminManager(s.Config)
	adminRepository := adminRepo.NewRepository(s.Factory.DB())

	server := httpServer.NewServer(s.Config)

	// Apply middleware
	server.Router().Use(middleware.SecurityHeaders(s.Config))
	server.Router().Use(middleware.CORS(s.Config))
	server.Router().Use(middleware.BodyLimit())
	server.Router().Use(middleware.RequireAuth(sessionManager))
	server.Router().Use(middleware.CheckMemberStatus(sessionManager, s.Factory.MembersProjection()))
	server.Router().Use(middleware.RateLimiter(sessionManager, s.Factory.SessionAnalyzer()))
	server.Router().Use(middleware.CSRFProtection())

	// Register handlers
	orgHandler := handlers.NewOrganizationHandler(s.Factory.OrganizationService(), s.Factory.OrganizationRatingsProjection(), sessionManager)
	orgHandler.RegisterRoutes(server.Router())

	authHandler := handlers.NewAuthHandler(s.Factory.MembersProjection(), s.Factory.FreightRequestsProjection(), s.Factory.OrganizationService(), sessionManager, s.Factory.SessionAnalyzer(), geoIPService)
	authHandler.RegisterRoutes(server.Router())

	adminHandler := handlers.NewAdminHandler(s.Factory.AdminService(), adminRepository, adminSessionManager, s.Factory.ReviewService(), s.Factory.ReviewsProjection(), s.Factory.FraudDataProjection())
	adminHandler.RegisterRoutes(server.Router())

	frHandler := handlers.NewFreightRequestHandler(s.Factory.FreightRequestService(), s.Factory.OrganizationService(), s.Factory.FreightRequestsProjection(), s.Factory.MembersProjection(), sessionManager)
	frHandler.RegisterRoutes(server.Router())

	historyHandler := handlers.NewHistoryHandler(s.Factory.HistoryService(), s.Factory.FreightRequestService(), sessionManager)
	historyHandler.RegisterRoutes(server.Router())

	geoHandler := handlers.NewGeoHandler(s.Factory.GeoProjection())
	geoHandler.RegisterRoutes(server.Router())

	notificationHandler := handlers.NewNotificationHandler(s.Factory.NotificationService(), sessionManager, s.Config)
	notificationHandler.RegisterRoutes(server.Router())

	subscriptionHandler := handlers.NewSubscriptionsHandler(s.Factory.FreightSubscriptionsProjection(), s.Factory.GeoProjection(), sessionManager)
	subscriptionHandler.RegisterRoutes(server.Router())

	supportHandler := handlers.NewSupportHandler(s.Factory.SupportService(), s.Factory.SupportTicketsProjection(), sessionManager)
	supportHandler.RegisterRoutes(server.Router())

	adminSupportHandler := handlers.NewAdminSupportHandler(s.Factory.SupportService(), s.Factory.SupportTicketsProjection(), adminSessionManager)
	adminSupportHandler.RegisterRoutes(server.Router())

	passwordResetHandler := handlers.NewPasswordResetHandler(
		s.Factory.MembersProjection(),
		s.Factory.PasswordResetProjection(),
		s.Factory.EmailTemplatesProjection(),
		s.Factory.EmailProvider(),
		s.Config,
	)
	passwordResetHandler.RegisterRoutes(server.Router())

	if s.Config.IsDevelopment() {
		devHandler := handlers.NewDevHandler(s.Config, s.Factory.MembersProjection(), s.Factory.OrganizationService(), sessionManager)
		devHandler.RegisterRoutes(server.Router())
	}

	s.server = server

	// Start server in goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		http.Serve(s.listener, server.Router())
	}()
}

func (s *Suite) waitForServer() error {
	client := &http.Client{Timeout: 100 * time.Millisecond}

	// Exponential backoff: 10ms -> 20ms -> 40ms -> ... -> 200ms max
	backoff := 10 * time.Millisecond
	maxBackoff := 200 * time.Millisecond
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		resp, err := client.Get(s.BaseURL + "/api/v1/geo/countries")
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff = min(backoff*2, maxBackoff)
		}
	}

	return fmt.Errorf("server did not become ready")
}

// Shutdown stops the test server and cleans up resources.
func (s *Suite) Shutdown() {
	if s.eventRouter != nil {
		if err := s.eventRouter.Close(); err != nil {
			slog.Error("failed to close event router", slog.String("error", err.Error()))
		}
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	if s.Factory != nil {
		s.Factory.Close()
	}
	if s.postgresContainer != nil {
		if err := s.postgresContainer.Stop(context.Background()); err != nil {
			slog.Error("failed to stop postgres container", slog.String("error", err.Error()))
		}
	}
}

// ShutdownShared stops the shared suite. Call this from TestMain.
func ShutdownShared() {
	sharedSuiteMu.Lock()
	defer sharedSuiteMu.Unlock()

	if sharedSuite != nil {
		sharedSuite.Shutdown()
		sharedSuite = nil
	}
}

// Sync waits for event handlers to process pending events.
// Uses a simple delay since watermill doesn't expose queue depth.
func (s *Suite) Sync() {
	time.Sleep(50 * time.Millisecond)
}
