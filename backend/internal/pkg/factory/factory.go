package factory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	adminApp "codeberg.org/udison/veziizi/backend/internal/application/admin"
	frApp "codeberg.org/udison/veziizi/backend/internal/application/freightrequest"
	historyApp "codeberg.org/udison/veziizi/backend/internal/application/history"
	"codeberg.org/udison/veziizi/backend/internal/application/history/display"
	notifApp "codeberg.org/udison/veziizi/backend/internal/application/notification"
	orderApp "codeberg.org/udison/veziizi/backend/internal/application/order"
	orgApp "codeberg.org/udison/veziizi/backend/internal/application/organization"
	reviewApp "codeberg.org/udison/veziizi/backend/internal/application/review"
	sessionApp "codeberg.org/udison/veziizi/backend/internal/application/session"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	frRules "codeberg.org/udison/veziizi/backend/internal/domain/notification/rules/freightrequest"
	orderRules "codeberg.org/udison/veziizi/backend/internal/domain/notification/rules/order"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/notifications"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/filestorage"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/sequence"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/pkg/config"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Factory is an IoC container that lazily creates and caches all dependencies.
// It accepts only *config.Config and creates everything else on demand.
// All getters are thread-safe with lazy initialization via sync.Once.
type Factory struct {
	cfg *config.Config

	// Base infrastructure (lazy)
	pool     *pgxpool.Pool
	poolOnce sync.Once
	poolErr  error

	txManager   dbtx.TxManager
	txOnce      sync.Once

	eventStore     eventstore.Store
	eventStoreOnce sync.Once

	publisher     *messaging.EventPublisher
	publisherOnce sync.Once
	publisherErr  error

	fileStorage     filestorage.FileStorage
	fileStorageOnce sync.Once

	telegramClient     *notifications.TelegramClient
	telegramClientOnce sync.Once

	// Services (lazy)
	orgService *orgApp.Service
	orgOnce    sync.Once

	adminService *adminApp.Service
	adminOnce    sync.Once

	frService *frApp.Service
	frOnce    sync.Once

	orderService *orderApp.Service
	orderOnce    sync.Once

	historyService *historyApp.Service
	historyOnce    sync.Once

	reviewService *reviewApp.Service
	reviewOnce    sync.Once

	notificationService *notifApp.Service
	notificationOnce    sync.Once

	// Projections (lazy)
	membersProjection *projections.MembersProjection
	membersOnce       sync.Once

	invitationsProjection *projections.InvitationsProjection
	invitationsOnce       sync.Once

	pendingOrgsProjection *projections.PendingOrganizationsProjection
	pendingOrgsOnce       sync.Once

	frProjection *projections.FreightRequestsProjection
	frProjOnce   sync.Once

	ordersProjection *projections.OrdersProjection
	ordersOnce       sync.Once

	ratingsProjection *projections.OrganizationRatingsProjection
	ratingsOnce       sync.Once

	fraudDataProjection *projections.FraudDataProjection
	fraudDataOnce       sync.Once

	reviewsProjection *projections.ReviewsProjection
	reviewsOnce       sync.Once

	orderFraudProjection *projections.OrderFraudProjection
	orderFraudOnce       sync.Once

	sessionFraudProjection *projections.SessionFraudProjection
	sessionFraudOnce       sync.Once

	organizationsProjection *projections.OrganizationsProjection
	organizationsOnce       sync.Once

	geoProjection *projections.GeoProjection
	geoOnce       sync.Once

	notificationPreferencesProjection *projections.NotificationPreferencesProjection
	notificationPreferencesOnce       sync.Once

	inappNotificationsProjection *projections.InAppNotificationsProjection
	inappNotificationsOnce       sync.Once

	deliveryLogProjection *projections.NotificationDeliveryLogProjection
	deliveryLogOnce       sync.Once

	telegramLinkProjection *projections.TelegramLinkProjection
	telegramLinkOnce       sync.Once

	// Проекция подписок на заявки (opt-in модель)
	freightSubscriptionsProjection *projections.FreightSubscriptionsProjection
	freightSubscriptionsOnce       sync.Once

	// Analyzers (lazy)
	reviewAnalyzer *reviewApp.Analyzer
	analyzerOnce   sync.Once

	sessionAnalyzer *sessionApp.SessionAnalyzer
	sessionOnce     sync.Once

	// Display registry (lazy)
	displayRegistry *display.Registry
	displayOnce     sync.Once

	// Sequence generator (lazy)
	seqGen     *sequence.Generator
	seqGenOnce sync.Once

	// Notification rules registry (lazy)
	notificationRulesRegistry *rules.Registry
	notificationRulesOnce     sync.Once
}

// New creates a new Factory IoC container.
// Only config is required - all other dependencies are created lazily on demand.
func New(cfg *config.Config) *Factory {
	return &Factory{
		cfg: cfg,
	}
}

// Config returns the application config
func (f *Factory) Config() *config.Config {
	return f.cfg
}

// Close gracefully shuts down all created resources.
// Should be called with defer in main().
func (f *Factory) Close() error {
	var errs []error

	// Close publisher if created
	if f.publisher != nil {
		if err := f.publisher.Close(); err != nil {
			slog.Error("failed to close publisher", slog.String("error", err.Error()))
			errs = append(errs, err)
		}
	}

	// Close pool if created
	if f.pool != nil {
		f.pool.Close()
	}

	if len(errs) > 0 {
		return fmt.Errorf("factory close errors: %v", errs)
	}
	return nil
}

// ============================================
// Base Infrastructure (lazy initialization)
// ============================================

// Pool returns the database connection pool (lazily created)
func (f *Factory) Pool() (*pgxpool.Pool, error) {
	f.poolOnce.Do(func() {
		ctx := context.Background()
		pool, err := pgxpool.New(ctx, f.cfg.Database.URL)
		if err != nil {
			f.poolErr = fmt.Errorf("create pool: %w", err)
			return
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			f.poolErr = fmt.Errorf("ping database: %w", err)
			return
		}

		f.pool = pool
		slog.Info("connected to database")
	})
	return f.pool, f.poolErr
}

// MustPool returns Pool or panics on error
func (f *Factory) MustPool() *pgxpool.Pool {
	pool, err := f.Pool()
	if err != nil {
		panic(fmt.Sprintf("failed to get pool: %v", err))
	}
	return pool
}

// DB returns the transaction manager (lazily created)
func (f *Factory) DB() dbtx.TxManager {
	f.txOnce.Do(func() {
		f.txManager = dbtx.NewTxExecutor(f.MustPool())
	})
	return f.txManager
}

// EventStore returns the event store (lazily created)
func (f *Factory) EventStore() eventstore.Store {
	f.eventStoreOnce.Do(func() {
		f.eventStore = eventstore.NewPostgresStore(f.DB())
	})
	return f.eventStore
}

// Publisher returns the event publisher (lazily created)
func (f *Factory) Publisher() (*messaging.EventPublisher, error) {
	f.publisherOnce.Do(func() {
		wmLogger := watermill.NewSlogLogger(slog.Default())
		publisher, err := messaging.NewEventPublisher(f.MustPool(), wmLogger)
		if err != nil {
			f.publisherErr = fmt.Errorf("create publisher: %w", err)
			return
		}
		f.publisher = publisher
	})
	return f.publisher, f.publisherErr
}

// MustPublisher returns Publisher or panics on error
func (f *Factory) MustPublisher() *messaging.EventPublisher {
	pub, err := f.Publisher()
	if err != nil {
		panic(fmt.Sprintf("failed to get publisher: %v", err))
	}
	return pub
}

// FileStorage returns file storage (lazily created)
func (f *Factory) FileStorage() filestorage.FileStorage {
	f.fileStorageOnce.Do(func() {
		f.fileStorage = filestorage.NewPostgresStorage(f.DB())
	})
	return f.fileStorage
}

// TelegramClient returns Telegram client (lazily created)
func (f *Factory) TelegramClient() *notifications.TelegramClient {
	f.telegramClientOnce.Do(func() {
		f.telegramClient = notifications.NewTelegramClient(f.cfg.Telegram.BotToken)
	})
	return f.telegramClient
}

// ============================================
// Services
// ============================================

func (f *Factory) OrganizationService() *orgApp.Service {
	f.orgOnce.Do(func() {
		f.orgService = orgApp.NewService(f.DB(), f.EventStore(), f.MustPublisher(), f.InvitationsProjection(), f.MembersProjection())
	})
	return f.orgService
}

func (f *Factory) AdminService() *adminApp.Service {
	f.adminOnce.Do(func() {
		f.adminService = adminApp.NewService(f.DB(), f.EventStore(), f.MustPublisher(), f.PendingOrganizationsProjection())
	})
	return f.adminService
}

func (f *Factory) FreightRequestService() *frApp.Service {
	f.frOnce.Do(func() {
		f.frService = frApp.NewService(f.DB(), f.EventStore(), f.MustPublisher(), f.SequenceGenerator())
	})
	return f.frService
}

func (f *Factory) OrderService() *orderApp.Service {
	f.orderOnce.Do(func() {
		f.orderService = orderApp.NewService(f.DB(), f.EventStore(), f.MustPublisher(), f.FileStorage(), f.SequenceGenerator())
	})
	return f.orderService
}

func (f *Factory) HistoryService() *historyApp.Service {
	f.historyOnce.Do(func() {
		f.historyService = historyApp.NewService(
			f.EventStore(),
			f.MembersProjection(),
			f.DisplayRegistry(),
		)
	})
	return f.historyService
}

func (f *Factory) ReviewService() *reviewApp.Service {
	f.reviewOnce.Do(func() {
		f.reviewService = reviewApp.NewService(f.DB(), f.EventStore(), f.MustPublisher())
	})
	return f.reviewService
}

func (f *Factory) NotificationService() *notifApp.Service {
	f.notificationOnce.Do(func() {
		f.notificationService = notifApp.NewService(
			f.NotificationPreferencesProjection(),
			f.InAppNotificationsProjection(),
			f.TelegramLinkProjection(),
		)
	})
	return f.notificationService
}

// ============================================
// Projections
// ============================================

func (f *Factory) MembersProjection() *projections.MembersProjection {
	f.membersOnce.Do(func() {
		f.membersProjection = projections.NewMembersProjection(f.DB())
	})
	return f.membersProjection
}

func (f *Factory) InvitationsProjection() *projections.InvitationsProjection {
	f.invitationsOnce.Do(func() {
		f.invitationsProjection = projections.NewInvitationsProjection(f.DB())
	})
	return f.invitationsProjection
}

func (f *Factory) PendingOrganizationsProjection() *projections.PendingOrganizationsProjection {
	f.pendingOrgsOnce.Do(func() {
		f.pendingOrgsProjection = projections.NewPendingOrganizationsProjection(f.DB())
	})
	return f.pendingOrgsProjection
}

func (f *Factory) FreightRequestsProjection() *projections.FreightRequestsProjection {
	f.frProjOnce.Do(func() {
		f.frProjection = projections.NewFreightRequestsProjection(f.DB())
	})
	return f.frProjection
}

func (f *Factory) OrdersProjection() *projections.OrdersProjection {
	f.ordersOnce.Do(func() {
		f.ordersProjection = projections.NewOrdersProjection(f.DB())
	})
	return f.ordersProjection
}

func (f *Factory) OrganizationRatingsProjection() *projections.OrganizationRatingsProjection {
	f.ratingsOnce.Do(func() {
		f.ratingsProjection = projections.NewOrganizationRatingsProjection(f.DB())
	})
	return f.ratingsProjection
}

func (f *Factory) FraudDataProjection() *projections.FraudDataProjection {
	f.fraudDataOnce.Do(func() {
		f.fraudDataProjection = projections.NewFraudDataProjection(f.DB())
	})
	return f.fraudDataProjection
}

func (f *Factory) ReviewsProjection() *projections.ReviewsProjection {
	f.reviewsOnce.Do(func() {
		f.reviewsProjection = projections.NewReviewsProjection(f.DB())
	})
	return f.reviewsProjection
}

func (f *Factory) OrderFraudProjection() *projections.OrderFraudProjection {
	f.orderFraudOnce.Do(func() {
		f.orderFraudProjection = projections.NewOrderFraudProjection(f.DB())
	})
	return f.orderFraudProjection
}

func (f *Factory) SessionFraudProjection() *projections.SessionFraudProjection {
	f.sessionFraudOnce.Do(func() {
		f.sessionFraudProjection = projections.NewSessionFraudProjection(f.DB())
	})
	return f.sessionFraudProjection
}

func (f *Factory) OrganizationsProjection() *projections.OrganizationsProjection {
	f.organizationsOnce.Do(func() {
		f.organizationsProjection = projections.NewOrganizationsProjection(f.DB())
	})
	return f.organizationsProjection
}

func (f *Factory) GeoProjection() *projections.GeoProjection {
	f.geoOnce.Do(func() {
		f.geoProjection = projections.NewGeoProjection(f.DB())
	})
	return f.geoProjection
}

func (f *Factory) NotificationPreferencesProjection() *projections.NotificationPreferencesProjection {
	f.notificationPreferencesOnce.Do(func() {
		f.notificationPreferencesProjection = projections.NewNotificationPreferencesProjection(f.DB())
	})
	return f.notificationPreferencesProjection
}

func (f *Factory) InAppNotificationsProjection() *projections.InAppNotificationsProjection {
	f.inappNotificationsOnce.Do(func() {
		f.inappNotificationsProjection = projections.NewInAppNotificationsProjection(f.DB())
	})
	return f.inappNotificationsProjection
}

func (f *Factory) DeliveryLogProjection() *projections.NotificationDeliveryLogProjection {
	f.deliveryLogOnce.Do(func() {
		f.deliveryLogProjection = projections.NewNotificationDeliveryLogProjection(f.DB())
	})
	return f.deliveryLogProjection
}

func (f *Factory) TelegramLinkProjection() *projections.TelegramLinkProjection {
	f.telegramLinkOnce.Do(func() {
		f.telegramLinkProjection = projections.NewTelegramLinkProjection(f.DB())
	})
	return f.telegramLinkProjection
}

// FreightSubscriptionsProjection возвращает проекцию подписок на заявки (opt-in модель)
func (f *Factory) FreightSubscriptionsProjection() *projections.FreightSubscriptionsProjection {
	f.freightSubscriptionsOnce.Do(func() {
		f.freightSubscriptionsProjection = projections.NewFreightSubscriptionsProjection(f.DB())
	})
	return f.freightSubscriptionsProjection
}

// ============================================
// Analyzers
// ============================================

func (f *Factory) ReviewAnalyzer() *reviewApp.Analyzer {
	f.analyzerOnce.Do(func() {
		f.reviewAnalyzer = reviewApp.NewAnalyzer(
			f.FraudDataProjection(),
			f.MembersProjection(),
		)
	})
	return f.reviewAnalyzer
}

func (f *Factory) SessionAnalyzer() *sessionApp.SessionAnalyzer {
	f.sessionOnce.Do(func() {
		f.sessionAnalyzer = sessionApp.NewSessionAnalyzer(f.SessionFraudProjection())
	})
	return f.sessionAnalyzer
}

func (f *Factory) DisplayRegistry() *display.Registry {
	f.displayOnce.Do(func() {
		f.displayRegistry = display.NewRegistry(
			f.MembersProjection(),
			f.OrganizationsProjection(),
		)
	})
	return f.displayRegistry
}

func (f *Factory) SequenceGenerator() *sequence.Generator {
	f.seqGenOnce.Do(func() {
		f.seqGen = sequence.NewGenerator(f.DB())
	})
	return f.seqGen
}

// NotificationRulesRegistry возвращает реестр правил уведомлений
func (f *Factory) NotificationRulesRegistry() *rules.Registry {
	f.notificationRulesOnce.Do(func() {
		f.notificationRulesRegistry = rules.NewRegistry()

		// Создаем зависимости для правил через адаптеры
		deps := rules.Dependencies{
			FreightRequests: rules.NewFreightRequestsAdapter(f.FreightRequestsProjection()),
			Orders:          rules.NewOrdersAdapter(f.OrdersProjection()),
			Members:         rules.NewMembersAdapter(f.MembersProjection()),
		}

		// Создаем matcher для подписок (opt-in модель)
		subscriptionMatcher := rules.NewFreightSubscriptionsAdapter(f.FreightSubscriptionsProjection())

		// Регистрируем правила FreightRequest
		f.notificationRulesRegistry.Register(frRules.NewOfferMadeRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferSelectedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferRejectedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferConfirmedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferDeclinedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferWithdrawnRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewFreightRequestCreatedRule(deps, subscriptionMatcher))

		// Регистрируем правила Order
		f.notificationRulesRegistry.Register(orderRules.NewOrderCreatedRule())
		f.notificationRulesRegistry.Register(orderRules.NewMessageSentRule(deps))
		f.notificationRulesRegistry.Register(orderRules.NewOrderCompletedRule(deps))
		f.notificationRulesRegistry.Register(orderRules.NewOrderCancelledRule(deps))
	})
	return f.notificationRulesRegistry
}
