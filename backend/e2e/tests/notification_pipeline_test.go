package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/udisondev/veziizi/backend/e2e/fixtures"
	"github.com/udisondev/veziizi/backend/e2e/setup"
	supportEvents "github.com/udisondev/veziizi/backend/internal/domain/support/events"
	eventHandlers "github.com/udisondev/veziizi/backend/internal/infrastructure/handlers"
	adminRepo "github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/admin"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// NotificationPipelineSuite покрывает notification-путь целиком:
// support-admin-notifier и notification-dispatcher (dedupGuard-хендлеры,
// контракт пропагации ошибок — см. dedup.go) + telegram-sender через
// NotificationMarshaler-топик. Полный продовый маршрут: HTTP → event store →
// outbox → forwarder → Redis Streams → consumer groups → NotificationBus →
// outbox → forwarder → notification.telegram → fake-телеграм.
type NotificationPipelineSuite struct {
	suite.Suite
	ts  *setup.Suite
	ctx *fixtures.TestContext
}

func TestNotificationPipelineSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(NotificationPipelineSuite))
}

func (s *NotificationPipelineSuite) SetupSuite() {
	s.ts = getSuite(s.T())
	s.ctx = fixtures.NewTestContext(s.T(), s.ts.BaseURL)
}

func (s *NotificationPipelineSuite) dedupRowCount(projection string, eventID *uuid.UUID) int {
	s.T().Helper()
	var count int
	query := `SELECT COUNT(*) FROM projection_event_dedup WHERE projection_name = $1`
	args := []any{projection}
	if eventID != nil {
		query += ` AND event_id = $2`
		args = append(args, *eventID)
	}
	err := s.ts.Factory.DB().QueryRow(context.Background(), query, args...).Scan(&count)
	s.Require().NoError(err, "count dedup rows")
	return count
}

// TestNP001_TicketCreatedNotifiesAdminTelegram: создание тикета через API
// доезжает до Telegram админа (fake) через support-admin-notifier →
// NotificationBus → telegram-sender, с dedup-резервом на пути.
func (s *NotificationPipelineSuite) TestNP001_TicketCreatedNotifiesAdminTelegram() {
	resp, err := s.ctx.Customer.Client.CreateTicket("Проблема с заявкой", "Не открывается список офферов")
	s.Require().NoError(err)
	s.Require().Less(resp.StatusCode, 300, string(resp.RawBody))

	s.ts.Sync()

	// Fake общий на suite — ассертим подмножеством, не абсолютным счётчиком.
	sent := s.ts.TelegramFake.SentTo(setup.TestAdminTelegramChatID)
	s.Require().NotEmpty(sent, "админ с telegram_chat_id должен получить уведомление о тикете")
	found := false
	for _, m := range sent {
		if strings.Contains(m.Text, "Новый тикет #") {
			found = true
			break
		}
	}
	s.Assert().True(found, "ожидали уведомление 'Новый тикет #', получили: %+v", sent)

	s.Assert().Positive(s.dedupRowCount("support-admin-notifier", nil),
		"dedup-резерв support-admin-notifier должен быть закоммичен")
}

// TestNP002_OfferMadeCreatesInAppNotification: OfferMade через полный пайплайн
// создаёт in-app уведомление заказчику (notification-dispatcher, legacy-путь,
// DefaultEnabledCategories: in-app включён без сидинга преференсов).
func (s *NotificationPipelineSuite) TestNP002_OfferMadeCreatesInAppNotification() {
	fr := fixtures.NewFreightRequest(s.T(), s.ctx.Customer.Client).Create()
	// Правило OfferMade читает freight_requests_lookup (другой consumer group):
	// дожидаемся проекции до создания оффера, иначе правило молча пропустит
	// событие без получателя.
	s.ts.Sync()

	fixtures.NewOffer(s.T(), s.ctx.Carrier.Client, fr.ID).Create()
	s.ts.Sync()

	var count int
	err := s.ts.Factory.DB().QueryRow(context.Background(),
		`SELECT COUNT(*) FROM inapp_notifications
		 WHERE entity_id = $1 AND member_id = $2 AND notification_type = 'new_offer'`,
		fr.ID, fr.MemberID,
	).Scan(&count)
	s.Require().NoError(err)
	s.Assert().Equal(1, count, "ровно одно in-app уведомление о новом оффере")
}

// TestNP003_DuplicateTicketCreatedDeliveredOnce: повторная at-least-once
// доставка TicketCreated (тот же event_id) применяется ровно один раз —
// dedupGuard-резерв подавляет второй вызов, админ получает одно сообщение.
// Хендлер вызывается напрямую (стиль at_least_once_test.go), публикация же
// едет дальше по реальному пайплайну до fake-телеграма.
func (s *NotificationPipelineSuite) TestNP003_DuplicateTicketCreatedDeliveredOnce() {
	f := s.ts.Factory
	h := eventHandlers.NewSupportAdminNotifierHandler(
		f.DB(),
		f.ProjectionEventDedupProjection(),
		adminRepo.NewRepository(f.DB()),
		f.MustNotificationBus(),
	)

	// Уникальный номер тикета — фильтр в общем fake'е.
	const ticketNumber int64 = 987654321
	evt := &supportEvents.TicketCreated{
		BaseEvent:        eventstore.NewBaseEvent(uuid.New(), supportEvents.AggregateType, 1),
		TicketNumber:     ticketNumber,
		MemberID:         uuid.New(),
		OrgID:            uuid.New(),
		Subject:          "dup delivery test",
		InitialMessageID: uuid.New(),
	}

	// Один event_id на обе доставки — как при redelivery из Redis Streams.
	eventID := uuid.New()
	mkCtx := func() context.Context {
		msg := message.NewMessage(uuid.NewString(), nil)
		msg.Metadata.Set("event_id", eventID.String())
		return cqrs.CtxWithOriginalMessage(context.Background(), msg)
	}

	s.Require().NoError(h.OnTicketCreated(mkCtx(), evt))
	s.Require().NoError(h.OnTicketCreated(mkCtx(), evt), "повторная доставка — молчаливый Ack")

	s.ts.Sync()

	matching := 0
	for _, m := range s.ts.TelegramFake.SentTo(setup.TestAdminTelegramChatID) {
		if strings.Contains(m.Text, "Новый тикет #987654321") {
			matching++
		}
	}
	s.Assert().Equal(1, matching, "дубль доставки не должен дать второе Telegram-сообщение")
	s.Assert().Equal(1, s.dedupRowCount("support-admin-notifier", &eventID),
		"ровно один dedup-резерв на event_id")
}
