package factory

import (
	"sync"

	adminApp "codeberg.org/udison/veziizi/backend/internal/application/admin"
	frApp "codeberg.org/udison/veziizi/backend/internal/application/freightrequest"
	historyApp "codeberg.org/udison/veziizi/backend/internal/application/history"
	"codeberg.org/udison/veziizi/backend/internal/application/history/display"
	orderApp "codeberg.org/udison/veziizi/backend/internal/application/order"
	orgApp "codeberg.org/udison/veziizi/backend/internal/application/organization"
	reviewApp "codeberg.org/udison/veziizi/backend/internal/application/review"
	sessionApp "codeberg.org/udison/veziizi/backend/internal/application/session"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/filestorage"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/sequence"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
)

// Factory creates and caches services and projections
// All getters are thread-safe with lazy initialization
type Factory struct {
	// Base dependencies
	db          dbtx.TxManager
	eventStore  eventstore.Store
	publisher   *messaging.EventPublisher
	fileStorage filestorage.FileStorage

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

	// Projections (lazy)
	membersProjection     *projections.MembersProjection
	membersOnce           sync.Once

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
}

// New creates a new Factory with base dependencies
func New(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	fileStorage filestorage.FileStorage,
) *Factory {
	return &Factory{
		db:          db,
		eventStore:  eventStore,
		publisher:   publisher,
		fileStorage: fileStorage,
	}
}

// Base dependencies getters

func (f *Factory) DB() dbtx.TxManager {
	return f.db
}

func (f *Factory) EventStore() eventstore.Store {
	return f.eventStore
}

func (f *Factory) Publisher() *messaging.EventPublisher {
	return f.publisher
}

func (f *Factory) FileStorage() filestorage.FileStorage {
	return f.fileStorage
}

// Services

func (f *Factory) OrganizationService() *orgApp.Service {
	f.orgOnce.Do(func() {
		f.orgService = orgApp.NewService(f.db, f.eventStore, f.publisher, f.InvitationsProjection(), f.MembersProjection())
	})
	return f.orgService
}

func (f *Factory) AdminService() *adminApp.Service {
	f.adminOnce.Do(func() {
		f.adminService = adminApp.NewService(f.db, f.eventStore, f.publisher, f.PendingOrganizationsProjection())
	})
	return f.adminService
}

func (f *Factory) FreightRequestService() *frApp.Service {
	f.frOnce.Do(func() {
		f.frService = frApp.NewService(f.db, f.eventStore, f.publisher, f.SequenceGenerator())
	})
	return f.frService
}

func (f *Factory) OrderService() *orderApp.Service {
	f.orderOnce.Do(func() {
		f.orderService = orderApp.NewService(f.db, f.eventStore, f.publisher, f.fileStorage, f.SequenceGenerator())
	})
	return f.orderService
}

func (f *Factory) HistoryService() *historyApp.Service {
	f.historyOnce.Do(func() {
		f.historyService = historyApp.NewService(
			f.eventStore,
			f.MembersProjection(),
			f.DisplayRegistry(),
		)
	})
	return f.historyService
}

func (f *Factory) ReviewService() *reviewApp.Service {
	f.reviewOnce.Do(func() {
		f.reviewService = reviewApp.NewService(f.db, f.eventStore, f.publisher)
	})
	return f.reviewService
}

// Projections

func (f *Factory) MembersProjection() *projections.MembersProjection {
	f.membersOnce.Do(func() {
		f.membersProjection = projections.NewMembersProjection(f.db)
	})
	return f.membersProjection
}

func (f *Factory) InvitationsProjection() *projections.InvitationsProjection {
	f.invitationsOnce.Do(func() {
		f.invitationsProjection = projections.NewInvitationsProjection(f.db)
	})
	return f.invitationsProjection
}

func (f *Factory) PendingOrganizationsProjection() *projections.PendingOrganizationsProjection {
	f.pendingOrgsOnce.Do(func() {
		f.pendingOrgsProjection = projections.NewPendingOrganizationsProjection(f.db)
	})
	return f.pendingOrgsProjection
}

func (f *Factory) FreightRequestsProjection() *projections.FreightRequestsProjection {
	f.frProjOnce.Do(func() {
		f.frProjection = projections.NewFreightRequestsProjection(f.db)
	})
	return f.frProjection
}

func (f *Factory) OrdersProjection() *projections.OrdersProjection {
	f.ordersOnce.Do(func() {
		f.ordersProjection = projections.NewOrdersProjection(f.db)
	})
	return f.ordersProjection
}

func (f *Factory) OrganizationRatingsProjection() *projections.OrganizationRatingsProjection {
	f.ratingsOnce.Do(func() {
		f.ratingsProjection = projections.NewOrganizationRatingsProjection(f.db)
	})
	return f.ratingsProjection
}

func (f *Factory) FraudDataProjection() *projections.FraudDataProjection {
	f.fraudDataOnce.Do(func() {
		f.fraudDataProjection = projections.NewFraudDataProjection(f.db)
	})
	return f.fraudDataProjection
}

func (f *Factory) ReviewsProjection() *projections.ReviewsProjection {
	f.reviewsOnce.Do(func() {
		f.reviewsProjection = projections.NewReviewsProjection(f.db)
	})
	return f.reviewsProjection
}

func (f *Factory) OrderFraudProjection() *projections.OrderFraudProjection {
	f.orderFraudOnce.Do(func() {
		f.orderFraudProjection = projections.NewOrderFraudProjection(f.db)
	})
	return f.orderFraudProjection
}

func (f *Factory) SessionFraudProjection() *projections.SessionFraudProjection {
	f.sessionFraudOnce.Do(func() {
		f.sessionFraudProjection = projections.NewSessionFraudProjection(f.db)
	})
	return f.sessionFraudProjection
}

func (f *Factory) OrganizationsProjection() *projections.OrganizationsProjection {
	f.organizationsOnce.Do(func() {
		f.organizationsProjection = projections.NewOrganizationsProjection(f.db)
	})
	return f.organizationsProjection
}

func (f *Factory) GeoProjection() *projections.GeoProjection {
	f.geoOnce.Do(func() {
		f.geoProjection = projections.NewGeoProjection(f.db)
	})
	return f.geoProjection
}

// Analyzers

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
		f.seqGen = sequence.NewGenerator(f.db)
	})
	return f.seqGen
}
