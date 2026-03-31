package factory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	adminApp "github.com/udisondev/veziizi/backend/internal/application/admin"
	frApp "github.com/udisondev/veziizi/backend/internal/application/freightrequest"
	historyApp "github.com/udisondev/veziizi/backend/internal/application/history"
	"github.com/udisondev/veziizi/backend/internal/application/history/display"
	notifApp "github.com/udisondev/veziizi/backend/internal/application/notification"
	orgApp "github.com/udisondev/veziizi/backend/internal/application/organization"
	reviewApp "github.com/udisondev/veziizi/backend/internal/application/review"
	sessionApp "github.com/udisondev/veziizi/backend/internal/application/session"
	supportApp "github.com/udisondev/veziizi/backend/internal/application/support"
	"github.com/udisondev/veziizi/backend/internal/domain/notification/rules"
	frRules "github.com/udisondev/veziizi/backend/internal/domain/notification/rules/freightrequest"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/adapters"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/notifications"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/filestorage"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/sequence"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
	"github.com/udisondev/veziizi/backend/internal/pkg/config"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
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

	emailProvider     notifications.EmailProvider
	emailProviderOnce sync.Once

	// Services (lazy)
	orgService *orgApp.Service
	orgOnce    sync.Once

	adminService *adminApp.Service
	adminOnce    sync.Once

	frService *frApp.Service
	frOnce    sync.Once

	historyService *historyApp.Service
	historyOnce    sync.Once

	reviewService *reviewApp.Service
	reviewOnce    sync.Once

	notificationService *notifApp.Service
	notificationOnce    sync.Once

	supportService *supportApp.Service
	supportOnce    sync.Once

	// Projections (lazy)
	membersProjection *projections.MembersProjection
	membersOnce       sync.Once

	invitationsProjection *projections.InvitationsProjection
	invitationsOnce       sync.Once

	pendingOrgsProjection *projections.PendingOrganizationsProjection
	pendingOrgsOnce       sync.Once

	frProjection *projections.FreightRequestsProjection
	frProjOnce   sync.Once

	ratingsProjection *projections.OrganizationRatingsProjection
	ratingsOnce       sync.Once

	fraudDataProjection *projections.FraudDataProjection
	fraudDataOnce       sync.Once

	reviewsProjection *projections.ReviewsProjection
	reviewsOnce       sync.Once

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

	// Проекция тикетов поддержки
	supportTicketsProjection *projections.SupportTicketsProjection
	supportTicketsOnce       sync.Once

	// Проекция email шаблонов
	emailTemplatesProjection *projections.EmailTemplatesProjection
	emailTemplatesOnce       sync.Once

	// Проекция токенов сброса пароля
	passwordResetProjection *projections.PasswordResetProjection
	passwordResetOnce       sync.Once

	// Проекция токенов верификации email
	emailVerificationProjection *projections.EmailVerificationProjection
	emailVerificationOnce       sync.Once

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
		// Таймаут на инициализацию подключения к БД
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

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

// EmailProvider returns Email provider (lazily created)
func (f *Factory) EmailProvider() notifications.EmailProvider {
	f.emailProviderOnce.Do(func() {
		f.emailProvider = notifications.NewEmailProvider(
			f.cfg.Email.Provider,
			f.cfg.Email.ResendAPIKey,
			f.cfg.Email.FromAddress,
			f.cfg.Email.FromName,
			f.cfg.Email.Enabled,
		)
	})
	return f.emailProvider
}

// ============================================
// Services
// ============================================

func (f *Factory) OrganizationService() *orgApp.Service {
	f.orgOnce.Do(func() {
		f.orgService = orgApp.NewService(f.DB(), f.EventStore(), f.MustPublisher(), f.InvitationsProjection(), f.MembersProjection(), f.OrganizationsProjection())
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
		memberChecker := adapters.NewMemberCheckerAdapter(f.OrganizationService())
		f.frService = frApp.NewService(f.DB(), f.EventStore(), f.MustPublisher(), f.SequenceGenerator(), memberChecker)
	})
	return f.frService
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
			f.DB(),
			f.NotificationPreferencesProjection(),
			f.InAppNotificationsProjection(),
			f.TelegramLinkProjection(),
			f.EmailVerificationProjection(),
			f.MustPublisher().RawPublisher(),
			f.cfg,
		)
	})
	return f.notificationService
}

func (f *Factory) SupportService() *supportApp.Service {
	f.supportOnce.Do(func() {
		f.supportService = supportApp.NewService(
			f.DB(),
			f.EventStore(),
			f.MustPublisher(),
			f.SequenceGenerator(),
		)
	})
	return f.supportService
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

// SupportTicketsProjection возвращает проекцию тикетов поддержки
func (f *Factory) SupportTicketsProjection() *projections.SupportTicketsProjection {
	f.supportTicketsOnce.Do(func() {
		f.supportTicketsProjection = projections.NewSupportTicketsProjection(f.DB())
	})
	return f.supportTicketsProjection
}

// EmailTemplatesProjection возвращает проекцию email шаблонов
func (f *Factory) EmailTemplatesProjection() *projections.EmailTemplatesProjection {
	f.emailTemplatesOnce.Do(func() {
		f.emailTemplatesProjection = projections.NewEmailTemplatesProjection(f.DB())
	})
	return f.emailTemplatesProjection
}

// PasswordResetProjection возвращает проекцию токенов сброса пароля
func (f *Factory) PasswordResetProjection() *projections.PasswordResetProjection {
	f.passwordResetOnce.Do(func() {
		f.passwordResetProjection = projections.NewPasswordResetProjection(f.DB())
	})
	return f.passwordResetProjection
}

// EmailVerificationProjection возвращает проекцию токенов верификации email
func (f *Factory) EmailVerificationProjection() *projections.EmailVerificationProjection {
	f.emailVerificationOnce.Do(func() {
		f.emailVerificationProjection = projections.NewEmailVerificationProjection(f.DB())
	})
	return f.emailVerificationProjection
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
			FreightRequests: adapters.NewFreightRequestsAdapter(f.FreightRequestsProjection()),
			Members:         adapters.NewMembersAdapter(f.MembersProjection()),
		}

		// Создаем matcher для подписок (opt-in модель)
		subscriptionMatcher := adapters.NewFreightSubscriptionsAdapter(f.FreightSubscriptionsProjection())

		// Регистрируем правила FreightRequest
		f.notificationRulesRegistry.Register(frRules.NewOfferMadeRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferSelectedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferRejectedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferConfirmedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferDeclinedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewOfferWithdrawnRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewFreightRequestCreatedRule(deps, subscriptionMatcher))
		f.notificationRulesRegistry.Register(frRules.NewFreightRequestCompletedRule(deps))
		f.notificationRulesRegistry.Register(frRules.NewCancelledAfterConfirmedRule(deps))
	})
	return f.notificationRulesRegistry
}
