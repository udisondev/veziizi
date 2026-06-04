package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/udisondev/veziizi/backend/e2e/helpers"
	freightEvents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	orgEvents "github.com/udisondev/veziizi/backend/internal/domain/organization/events"
	orgValues "github.com/udisondev/veziizi/backend/internal/domain/organization/values"
	eventHandlers "github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/pkg/factory"
)

// AtLeastOnceSuite проверяет устойчивость хендлеров к семантике Redis Streams
// consumer groups: повторная доставка (at-least-once) и нарушение порядка
// событий одного агрегата при N инстансах воркера. Хендлеры вызываются
// напрямую (без транспорта) — нас интересует контракт обработки, а не доставка,
// которая покрыта остальными e2e через полный пайплайн.
type AtLeastOnceSuite struct {
	suite.Suite
	f *factory.Factory
}

func TestAtLeastOnceSuite(t *testing.T) {
	suite.Run(t, new(AtLeastOnceSuite))
}

func (s *AtLeastOnceSuite) SetupSuite() {
	testSuite := getSuite(s.T())
	s.f = testSuite.Factory
}

func (s *AtLeastOnceSuite) memberRow(memberID uuid.UUID) (role, status string, roleVersion, statusVersion int64) {
	s.T().Helper()
	err := s.f.DB().QueryRow(context.Background(),
		`SELECT role, status, role_version, status_version FROM members_lookup WHERE id = $1`, memberID,
	).Scan(&role, &status, &roleVersion, &statusVersion)
	s.Require().NoError(err, "load member row")
	return role, status, roleVersion, statusVersion
}

func (s *AtLeastOnceSuite) memberRowExists(memberID uuid.UUID) bool {
	s.T().Helper()
	var exists bool
	err := s.f.DB().QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM members_lookup WHERE id = $1)`, memberID,
	).Scan(&exists)
	s.Require().NoError(err, "check member row")
	return exists
}

// TestALO001_StatusUpdateBeforeCreatedRetries: статусное событие, пришедшее
// раньше Created (конкурентный инстанс ещё не вставил строку), должно вернуть
// ошибку — retry middleware повторит, когда строка появится.
func (s *AtLeastOnceSuite) TestALO001_StatusUpdateBeforeCreatedRetries() {
	ctx := context.Background()
	h := eventHandlers.NewMembersHandler(s.f.DB())

	orgID := uuid.New()
	memberID := uuid.New()

	roleChanged := &orgEvents.MemberRoleChanged{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 3),
		MemberID:  memberID,
		OldRole:   orgValues.MemberRoleEmployee,
		NewRole:   orgValues.MemberRoleAdministrator,
		ChangedBy: uuid.New(),
	}

	// Строки ещё нет — out-of-order, ждём ошибку (сигнал retry).
	err := h.OnMemberRoleChanged(ctx, roleChanged)
	s.Require().Error(err, "событие раньше Created должно уходить в retry, а не теряться")

	// Created догнал — вставляем строку (v2).
	added := &orgEvents.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 2),
		MemberID:     memberID,
		Email:        helpers.RandomEmail(),
		PasswordHash: "hash",
		Name:         "Test Member",
		Role:         orgValues.MemberRoleEmployee,
	}
	s.Require().NoError(h.OnMemberAdded(ctx, added))

	// Retry статусного события теперь проходит.
	s.Require().NoError(h.OnMemberRoleChanged(ctx, roleChanged))

	role, _, roleVersion, _ := s.memberRow(memberID)
	s.Assert().Equal("administrator", role)
	s.Assert().Equal(int64(3), roleVersion)
}

// TestALO002_StaleEventDoesNotOverwriteNewerState: устаревшее событие (меньшая
// версия агрегата) не должно перетирать более свежий статус — молчаливый Ack.
func (s *AtLeastOnceSuite) TestALO002_StaleEventDoesNotOverwriteNewerState() {
	ctx := context.Background()
	h := eventHandlers.NewMembersHandler(s.f.DB())

	orgID := uuid.New()
	memberID := uuid.New()

	added := &orgEvents.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 1),
		MemberID:     memberID,
		Email:        helpers.RandomEmail(),
		PasswordHash: "hash",
		Name:         "Test Member",
		Role:         orgValues.MemberRoleEmployee,
	}
	s.Require().NoError(h.OnMemberAdded(ctx, added))

	// Свежее событие (v5): блокировка.
	blocked := &orgEvents.MemberBlocked{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 5),
		MemberID:  memberID,
		BlockedBy: uuid.New(),
	}
	s.Require().NoError(h.OnMemberBlocked(ctx, blocked))

	// Устаревшее событие (v4, другой инстанс отстал): разблокировка.
	// При реальном порядке v4 < v5 разблокировка произошла РАНЬШЕ блокировки —
	// итоговое состояние должно остаться blocked.
	unblocked := &orgEvents.MemberUnblocked{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 4),
		MemberID:  memberID,
	}
	s.Require().NoError(h.OnMemberUnblocked(ctx, unblocked), "устаревшее событие должно молча ack'аться")

	_, status, _, statusVersion := s.memberRow(memberID)
	s.Assert().Equal("blocked", status, "устаревший unblock не должен перетирать свежий block")
	s.Assert().Equal(int64(5), statusVersion)
}

// TestALO003_DuplicateDeliveryIsIdempotent: повторная доставка того же события
// (тот же payload, та же версия) применяется ровно один раз и не ошибается.
func (s *AtLeastOnceSuite) TestALO003_DuplicateDeliveryIsIdempotent() {
	ctx := context.Background()
	h := eventHandlers.NewMembersHandler(s.f.DB())

	orgID := uuid.New()
	memberID := uuid.New()

	added := &orgEvents.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 1),
		MemberID:     memberID,
		Email:        helpers.RandomEmail(),
		PasswordHash: "hash",
		Name:         "Test Member",
		Role:         orgValues.MemberRoleEmployee,
	}
	s.Require().NoError(h.OnMemberAdded(ctx, added))
	s.Require().NoError(h.OnMemberAdded(ctx, added), "повтор Created — no-op")

	blocked := &orgEvents.MemberBlocked{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 2),
		MemberID:  memberID,
		BlockedBy: uuid.New(),
	}
	s.Require().NoError(h.OnMemberBlocked(ctx, blocked))
	s.Require().NoError(h.OnMemberBlocked(ctx, blocked), "повтор статусного события — идемпотентен")

	_, status, _, statusVersion := s.memberRow(memberID)
	s.Assert().Equal("blocked", status)
	s.Assert().Equal(int64(2), statusVersion)
}

// TestALO005_OrthogonalFieldsOutOfOrder: role и status guard'ятся раздельными
// version-колонками. Свежий status-событие (v5), обогнавшее более раннее
// role-событие (v3), НЕ должно заставить guard выбросить роль — v3 несёт
// единственную актуальную роль.
func (s *AtLeastOnceSuite) TestALO005_OrthogonalFieldsOutOfOrder() {
	ctx := context.Background()
	h := eventHandlers.NewMembersHandler(s.f.DB())

	orgID := uuid.New()
	memberID := uuid.New()

	added := &orgEvents.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 1),
		MemberID:     memberID,
		Email:        helpers.RandomEmail(),
		PasswordHash: "hash",
		Name:         "Test Member",
		Role:         orgValues.MemberRoleEmployee,
	}
	s.Require().NoError(h.OnMemberAdded(ctx, added))

	// Out-of-order: блокировка (v5) обогнала смену роли (v3).
	blocked := &orgEvents.MemberBlocked{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 5),
		MemberID:  memberID,
		BlockedBy: uuid.New(),
	}
	s.Require().NoError(h.OnMemberBlocked(ctx, blocked))

	roleChanged := &orgEvents.MemberRoleChanged{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 3),
		MemberID:  memberID,
		OldRole:   orgValues.MemberRoleEmployee,
		NewRole:   orgValues.MemberRoleAdministrator,
		ChangedBy: uuid.New(),
	}
	s.Require().NoError(h.OnMemberRoleChanged(ctx, roleChanged))

	role, status, roleVersion, statusVersion := s.memberRow(memberID)
	s.Assert().Equal("administrator", role, "роль из v3 — единственная актуальная, v5-блокировка не должна её выбрасывать")
	s.Assert().Equal("blocked", status)
	s.Assert().Equal(int64(3), roleVersion)
	s.Assert().Equal(int64(5), statusVersion)
}

// TestALO006_RemovedMemberNotResurrected: tombstone в members_removed защищает
// от воскрешения строки повторно доставленным MemberAdded и переводит статусные
// события удалённого участника в молчаливый Ack вместо вечного retry → DLQ.
func (s *AtLeastOnceSuite) TestALO006_RemovedMemberNotResurrected() {
	ctx := context.Background()
	h := eventHandlers.NewMembersHandler(s.f.DB())

	orgID := uuid.New()
	memberID := uuid.New()

	added := &orgEvents.MemberAdded{
		BaseEvent:    eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 2),
		MemberID:     memberID,
		Email:        helpers.RandomEmail(),
		PasswordHash: "hash",
		Name:         "Test Member",
		Role:         orgValues.MemberRoleEmployee,
	}
	s.Require().NoError(h.OnMemberAdded(ctx, added))

	removed := &orgEvents.MemberRemoved{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 8),
		MemberID:  memberID,
	}
	s.Require().NoError(h.OnMemberRemoved(ctx, removed))
	s.Require().False(s.memberRowExists(memberID), "строка удалена")

	// Повторная доставка Added после Removed — НЕ воскрешает строку.
	s.Require().NoError(h.OnMemberAdded(ctx, added))
	s.Assert().False(s.memberRowExists(memberID), "redelivered Added не должен воскрешать удалённого участника")

	// Статусное событие после удаления — молчаливый Ack (не retry/DLQ).
	roleChanged := &orgEvents.MemberRoleChanged{
		BaseEvent: eventstore.NewBaseEvent(orgID, orgEvents.AggregateType, 6),
		MemberID:  memberID,
		OldRole:   orgValues.MemberRoleEmployee,
		NewRole:   orgValues.MemberRoleAdministrator,
		ChangedBy: uuid.New(),
	}
	s.Require().NoError(h.OnMemberRoleChanged(ctx, roleChanged),
		"статусное событие удалённого участника должно ack'аться по tombstone'у")
	s.Assert().False(s.memberRowExists(memberID))
}

// TestALO004_DuplicateReviewLeftCreatesSingleReview: повторная доставка
// ReviewLeft не создаёт второй Review — ReviewID детерминирован, конфликт
// версий в event store трактуется как идемпотентный повтор (Ack).
func (s *AtLeastOnceSuite) TestALO004_DuplicateReviewLeftCreatesSingleReview() {
	ctx := context.Background()
	h := eventHandlers.NewReviewReceiverHandler(s.f.ReviewService())

	reviewID := uuid.New()
	evt := &freightEvents.ReviewLeft{
		BaseEvent:        eventstore.NewBaseEvent(uuid.New(), freightEvents.AggregateType, 7),
		ReviewID:         reviewID,
		ReviewerOrgID:    uuid.New(),
		ReviewerMemberID: uuid.New(),
		ReviewedOrgID:    uuid.New(),
		Rating:           5,
		Comment:          "great",
		FreightAmount:    1_000_000,
		FreightCurrency:  "RUB",
		FreightCreatedAt: time.Now().Add(-48 * time.Hour).Unix(),
		CompletedAt:      time.Now().Add(-24 * time.Hour).Unix(),
	}

	s.Require().NoError(h.OnReviewLeft(ctx, evt))
	s.Require().NoError(h.OnReviewLeft(ctx, evt), "повторная доставка ReviewLeft — Ack, не вечный retry")

	// Review существует ровно один: повтор упёрся в UNIQUE(aggregate_id, version).
	review, err := s.f.ReviewService().Get(ctx, reviewID)
	s.Require().NoError(err)
	s.Assert().Equal(5, review.Rating())
	s.Assert().Equal(int64(1), review.Version(), "повтор не должен дописывать события в агрегат")
}
