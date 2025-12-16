package factory

import (
	"sync"

	adminApp "codeberg.org/udison/veziizi/backend/internal/application/admin"
	frApp "codeberg.org/udison/veziizi/backend/internal/application/freightrequest"
	orderApp "codeberg.org/udison/veziizi/backend/internal/application/order"
	orgApp "codeberg.org/udison/veziizi/backend/internal/application/organization"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/messaging"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/filestorage"
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
		f.orgService = orgApp.NewService(f.db, f.eventStore, f.publisher, f.InvitationsProjection())
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
		f.frService = frApp.NewService(f.db, f.eventStore, f.publisher)
	})
	return f.frService
}

func (f *Factory) OrderService() *orderApp.Service {
	f.orderOnce.Do(func() {
		f.orderService = orderApp.NewService(f.db, f.eventStore, f.publisher, f.fileStorage)
	})
	return f.orderService
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
