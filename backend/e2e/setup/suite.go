// Package setup provides E2E test infrastructure for the veziizi API.
// It handles server startup, database management, and test lifecycle.
package setup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/notification/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/review/events"
	_ "github.com/udisondev/veziizi/backend/internal/domain/support/events"

	"github.com/ThreeDotsLabs/watermill"
	wmSql "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	eventHandlers "github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	adminRepo "github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	httpServer "github.com/udisondev/veziizi/backend/internal/interfaces/http"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/handlers"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/middleware"
	"github.com/udisondev/veziizi/backend/internal/interfaces/http/session"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
	"github.com/udisondev/veziizi/backend/internal/pkg/geoip"
)

// e2eForwarderGroup — consumer group SQL-подписчика outbox-топика в suite.
const e2eForwarderGroup = "e2e_forwarder"

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
	// consumerGroups: topic → e2e consumer group'ы, зарегистрированные на нём.
	// Используется Sync() для ожидания опустошения стримов.
	consumerGroups map[string][]string
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
		consumerGroups:    sharedSuite.consumerGroups,
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

// NewSuiteUnmanaged creates an isolated test suite without binding its
// lifecycle to t. Callers are responsible for invoking Shutdown() themselves
// (typically from TestMain when running per-test-class shared suites).
func NewSuiteUnmanaged(t *testing.T) *Suite {
	suite, err := newSuite(t)
	if err != nil {
		t.Fatalf("failed to create suite: %v", err)
	}
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

	// Reuse a process-wide Postgres container; each suite gets its own
	// database inside it. Much faster than spawning a container per suite
	// (no docker daemon contention, no per-suite startup wait).
	if _, err := GetSharedPostgres(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start shared postgres: %w", err)
	}
	dsn, err := CreateTestDatabase(ctx, t.Name())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create per-suite database: %w", err)
	}

	// Shared Redis container; each suite gets its own DB index — нативная
	// изоляция стримов и consumer group'ов между сьютами.
	if _, err := GetSharedRedis(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start shared redis: %w", err)
	}
	redisURL, err := NextRedisURL()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to allocate redis db: %w", err)
	}

	cfg := testConfigWithDSN(dsn, redisURL)

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
		T:              t,
		BaseURL:        baseURL,
		Factory:        f,
		Config:         cfg,
		listener:       listener,
		ctx:            ctx,
		cancel:         cancel,
		consumerGroups: make(map[string][]string),
		// postgresContainer left nil: lifecycle owned by GetSharedPostgres /
		// StopSharedPostgres (called from TestMain).
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

// startEventHandlers поднимает in-process эквивалент production-пайплайна:
//   - forwarder: SQL-подписчик outbox-топика (50ms poll) → Redis-стримы
//     этого suite (свой DB index);
//   - CQRS-EventGroupProcessor для каждого воркера на Redis consumer group'ах
//     с e2e_-префиксами.
//
// Это полный продовый путь доставки: publish → Postgres outbox → forwarder →
// Redis Streams → consumer groups → хендлеры.
func (s *Suite) startEventHandlers() error {
	wmLogger := watermill.NewSlogLogger(slog.Default())
	pool := s.Factory.MustPool()
	db := s.Factory.DB()
	redisClient := s.Factory.MustRedisClient()

	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return fmt.Errorf("failed to create router: %w", err)
	}

	// In-process forwarder: переиспользует общий router (forwarder.Config.Router),
	// поэтому отдельный Run не нужен — стартует вместе с router.Run ниже.
	outboxSub, err := wmSql.NewSubscriber(
		wmSql.BeginnerFromPgx(pool),
		wmSql.SubscriberConfig{
			SchemaAdapter:  wmSql.DefaultPostgreSQLSchema{},
			OffsetsAdapter: wmSql.DefaultPostgreSQLOffsetsAdapter{},
			ConsumerGroup:  e2eForwarderGroup,
			PollInterval:   50 * time.Millisecond,
		},
		wmLogger,
	)
	if err != nil {
		return fmt.Errorf("create outbox subscriber: %w", err)
	}
	if err := outboxSub.SubscribeInitialize(messaging.OutboxTopic); err != nil {
		return fmt.Errorf("init outbox subscriber: %w", err)
	}
	redisPub, err := messaging.NewRedisPublisher(redisClient, s.Config.Redis, wmLogger)
	if err != nil {
		return fmt.Errorf("create redis publisher: %w", err)
	}
	if _, err := forwarder.NewForwarder(outboxSub, redisPub, wmLogger, forwarder.Config{
		ForwarderTopic: messaging.OutboxTopic,
		Router:         router,
	}); err != nil {
		return fmt.Errorf("create forwarder: %w", err)
	}

	// Helper для регистрации CQRS-handler'ов одного воркера: Redis consumer
	// group + suite-уникальное имя консьюмера.
	register := func(topic, consumerGroup string, handlers ...cqrs.GroupEventHandler) error {
		ep, err := cqrs.NewEventGroupProcessorWithConfig(router, cqrs.EventGroupProcessorConfig{
			GenerateSubscribeTopic: func(cqrs.EventGroupProcessorGenerateSubscribeTopicParams) (string, error) {
				return topic, nil
			},
			SubscriberConstructor: func(cqrs.EventGroupProcessorSubscriberConstructorParams) (message.Subscriber, error) {
				return messaging.NewRedisSubscriber(redisClient, consumerGroup, "e2e-"+consumerGroup, s.Config.Redis, wmLogger)
			},
			Marshaler:         messaging.EventEnvelopeMarshaler{},
			Logger:            wmLogger,
			AckOnUnknownEvent: true,
		})
		if err != nil {
			return fmt.Errorf("create event processor %s: %w", consumerGroup, err)
		}
		s.consumerGroups[topic] = append(s.consumerGroups[topic], consumerGroup)
		return ep.AddHandlersGroup(consumerGroup, handlers...)
	}

	// Списки хендлеров берутся из eventHandlers.XGroupHandlers — те же функции,
	// что используют cmd/workers/*/main.go: e2e гоняет ровно прод-набор, и новый
	// OnXxx нельзя забыть в одной из копий.
	membersH := eventHandlers.NewMembersHandler(db)
	if err := register("organization.events", "e2e_members",
		eventHandlers.MembersGroupHandlers(membersH)...); err != nil {
		return err
	}

	orgsH := eventHandlers.NewOrganizationsHandler(s.Factory.EventStore(), s.Factory.OrganizationsProjection(), s.Factory.FreightRequestsProjection())
	if err := register("organization.events", "e2e_organizations",
		eventHandlers.OrganizationsGroupHandlers(orgsH)...); err != nil {
		return err
	}

	invH := eventHandlers.NewInvitationsHandler(db)
	if err := register("organization.events", "e2e_invitations",
		eventHandlers.InvitationsGroupHandlers(invH)...); err != nil {
		return err
	}

	pendingH := eventHandlers.NewPendingOrganizationsHandler(db)
	if err := register("organization.events", "e2e_pending_orgs",
		eventHandlers.PendingOrganizationsGroupHandlers(pendingH)...); err != nil {
		return err
	}

	frH := eventHandlers.NewFreightRequestsHandler(db, s.Factory.EventStore(), s.Factory.FreightInvitesProjection())
	if err := register("freightrequest.events", "e2e_freight_requests",
		eventHandlers.FreightRequestsGroupHandlers(frH)...); err != nil {
		return err
	}

	supportH := eventHandlers.NewSupportTicketsHandler(db)
	if err := register("support.events", "e2e_support_tickets",
		eventHandlers.SupportTicketsGroupHandlers(supportH)...); err != nil {
		return err
	}

	fraudH := eventHandlers.NewFraudsterHandler(s.Factory.ReviewService(), s.Factory.ReviewsProjection(), s.Factory.FraudDataProjection())
	if err := register("organization.events", "e2e_fraudster",
		eventHandlers.FraudsterGroupHandlers(fraudH)...); err != nil {
		return err
	}

	receiverH := eventHandlers.NewReviewReceiverHandler(s.Factory.ReviewService())
	if err := register("freightrequest.events", "e2e_review_receiver",
		eventHandlers.ReviewReceiverGroupHandlers(receiverH)...); err != nil {
		return err
	}

	analyzerH := eventHandlers.NewReviewAnalyzerHandler(s.Factory.ReviewService(), s.Factory.ReviewAnalyzer())
	if err := register("review.events", "e2e_review_analyzer",
		eventHandlers.ReviewAnalyzerGroupHandlers(analyzerH)...); err != nil {
		return err
	}

	reviewsProjH := eventHandlers.NewReviewsProjectionHandler(
		s.Factory.DB(),
		s.Factory.FraudDataProjection(),
		s.Factory.OrganizationRatingsProjection(),
		s.Factory.ProjectionEventDedupProjection(),
	)
	if err := register("review.events", "e2e_reviews_projection",
		eventHandlers.ReviewsProjectionGroupHandlers(reviewsProjH)...); err != nil {
		return err
	}

	vehiclesH := eventHandlers.NewVehiclesHandler(db, s.Factory.VehiclesProjection(), s.Factory.PendingVehiclesProjection())
	if err := register("organization.events", "e2e_vehicles",
		eventHandlers.VehiclesGroupHandlers(vehiclesH)...); err != nil {
		return err
	}

	s.eventRouter = router

	// Start router in background
	s.wg.Go(func() {
		if err := router.Run(s.ctx); err != nil {
			slog.Error("event router error", slog.String("error", err.Error()))
		}
	})

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

	// Health endpoints — без auth/CSRF/rate limiter
	healthHandler := handlers.NewHealthHandler(s.Factory.MustPool())
	server.Router().Group(func(r chi.Router) {
		healthHandler.RegisterRoutes(r)
	})

	// API routes с полным middleware stack
	server.Router().Group(func(r chi.Router) {
		r.Use(middleware.SecurityHeaders(s.Config))
		r.Use(middleware.CORS(s.Config))
		r.Use(middleware.BodyLimit(s.Config))
		r.Use(middleware.RequireAuth(sessionManager))
		r.Use(middleware.CheckMemberStatus(sessionManager, s.Factory.MembersProjection()))
		r.Use(middleware.RateLimiter(sessionManager, s.Factory.SessionAnalyzer()))
		r.Use(middleware.CSRFProtection())

		orgHandler := handlers.NewOrganizationHandler(s.Factory.OrganizationService(), s.Factory.OrganizationRatingsProjection(), s.Factory.FreightRequestsProjection(), sessionManager)
		orgHandler.RegisterRoutes(r)

		authHandler := handlers.NewAuthHandler(s.Factory.MembersProjection(), s.Factory.FreightRequestsProjection(), s.Factory.OrganizationService(), sessionManager, s.Factory.SessionAnalyzer(), geoIPService)
		authHandler.RegisterRoutes(r)

		adminHandler := handlers.NewAdminHandler(s.Factory.AdminService(), adminRepository, adminSessionManager, s.Factory.ReviewService(), s.Factory.ReviewsProjection(), s.Factory.FraudDataProjection())
		r.Route("/api/v1/admin", func(r chi.Router) {
			r.Use(middleware.RequireAdminAuth(adminSessionManager))
			adminHandler.RegisterRoutes(r)

			adminSupportHandler := handlers.NewAdminSupportHandler(s.Factory.SupportService(), s.Factory.SupportTicketsProjection(), adminSessionManager)
			adminSupportHandler.RegisterRoutes(r)
		})

		frHandler := handlers.NewFreightRequestHandler(s.Factory.FreightRequestService(), s.Factory.OrganizationService(), s.Factory.FreightRequestsProjection(), s.Factory.MembersProjection(), s.Factory.FreightInvitesProjection(), s.Factory.FreightRequestViewsProjection(), sessionManager)
		frHandler.RegisterRoutes(r)

		historyHandler := handlers.NewHistoryHandler(s.Factory.HistoryService(), s.Factory.FreightRequestService(), sessionManager)
		historyHandler.RegisterRoutes(r)

		geoHandler := handlers.NewGeoHandler(s.Factory.GeoProjection())
		geoHandler.RegisterRoutes(r)

		notificationHandler := handlers.NewNotificationHandler(s.Factory.NotificationService(), sessionManager, s.Config)
		notificationHandler.RegisterRoutes(r)

		subscriptionHandler := handlers.NewSubscriptionsHandler(s.Factory.FreightSubscriptionsProjection(), s.Factory.GeoProjection(), sessionManager)
		subscriptionHandler.RegisterRoutes(r)

		supportHandler := handlers.NewSupportHandler(s.Factory.SupportService(), s.Factory.SupportTicketsProjection(), sessionManager)
		supportHandler.RegisterRoutes(r)

		passwordResetHandler := handlers.NewPasswordResetHandler(
			s.Factory.MembersProjection(),
			s.Factory.PasswordResetProjection(),
			s.Factory.EmailTemplatesProjection(),
			s.Factory.EmailProvider(),
			s.Config,
		)
		passwordResetHandler.RegisterRoutes(r)

		if s.Config.IsDevelopment() {
			r.Route("/api/v1/dev", func(r chi.Router) {
				r.Use(middleware.DevOnly(s.Config))
				devHandler := handlers.NewDevHandler(s.Config, s.Factory.MembersProjection(), s.Factory.OrganizationService(), sessionManager)
				devHandler.RegisterRoutesWithRouter(r)
			})
		}
	})

	s.server = server

	// Start server in goroutine
	s.wg.Go(func() {
		if err := http.Serve(s.listener, server.Router()); err != nil {
			slog.Error("http serve error", slog.String("error", err.Error()))
		}
	})
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
			_ = resp.Body.Close()
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
		if err := s.listener.Close(); err != nil {
			slog.Error("failed to close listener", slog.String("error", err.Error()))
		}
	}
	s.wg.Wait()
	if s.Factory != nil {
		if err := s.Factory.Close(); err != nil {
			slog.Error("failed to close factory", slog.String("error", err.Error()))
		}
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

// Sync ждёт, пока пайплайн доставки прогонит все опубликованные события:
//  1. forwarder вычитал Postgres outbox до конца (offset_acked == MAX(offset));
//  2. каждая e2e consumer group на каждом стриме догнала last stream id и не
//     имеет pending (недоack'нутых) сообщений.
//
// Фиксированный sleep здесь больше не работает: к доставке добавился хоп
// outbox → forwarder → Redis. Поллим состояние с дедлайном; по истечении
// возвращаемся молча — тест упадёт сам с внятным diff'ом состояния проекции.
func (s *Suite) Sync() {
	const (
		deadline = 5 * time.Second
		step     = 15 * time.Millisecond
	)

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	pool := s.Factory.MustPool()
	redisClient := s.Factory.MustRedisClient()

	// Два стабильных прохода подряд: одиночный снапшот может поймать паузу
	// между «forwarder вычитал outbox» и «сообщение появилось в стриме»
	// (forwarder ack'ает outbox после publish, но XADD и offset-commit — две
	// системы), а также окно между ack хендлера и публикацией им нового
	// события (review-receiver порождает review.events).
	stable := 0
	for stable < 2 {
		if ctx.Err() != nil {
			slog.Warn("suite sync deadline exceeded, continuing anyway")
			return
		}
		if s.pipelineDrained(ctx, pool, redisClient) {
			stable++
		} else {
			stable = 0
		}
		time.Sleep(step)
	}
}

func (s *Suite) pipelineDrained(ctx context.Context, pool *pgxpool.Pool, redisClient redis.UniversalClient) bool {
	// 1. Forwarder догнал outbox.
	var maxOffset, acked int64
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(MAX("offset"), 0) FROM "watermill_`+messaging.OutboxTopic+`"`,
	).Scan(&maxOffset); err != nil {
		return false
	}
	if err := pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(offset_acked), 0) FROM "watermill_offsets_`+messaging.OutboxTopic+`" WHERE consumer_group = $1`,
		e2eForwarderGroup,
	).Scan(&acked); err != nil {
		return false
	}
	if acked < maxOffset {
		return false
	}

	// 2. Все consumer group'ы догнали свои стримы.
	for topic, groups := range s.consumerGroups {
		lastID, err := redisClient.XInfoStream(ctx, topic).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || strings.Contains(err.Error(), "no such key") {
				continue // стрим ещё не создан — нечего обрабатывать
			}
			return false
		}
		groupInfos, err := redisClient.XInfoGroups(ctx, topic).Result()
		if err != nil {
			return false
		}
		byName := make(map[string]redis.XInfoGroup, len(groupInfos))
		for _, g := range groupInfos {
			byName[g.Name] = g
		}
		for _, group := range groups {
			g, ok := byName[group]
			if !ok {
				return false // группа ещё не создана подписчиком
			}
			if g.Pending > 0 || g.LastDeliveredID != lastID.LastGeneratedID {
				return false
			}
		}
	}

	return true
}
