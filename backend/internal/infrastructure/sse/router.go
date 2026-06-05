package sse

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	frevents "github.com/udisondev/veziizi/backend/internal/domain/freightrequest/events"
	notifevents "github.com/udisondev/veziizi/backend/internal/domain/notification/events"
	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	supportevents "github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// Узкие интерфейсы поверх проекций — роутеру нужны только адресаты события.
// Конкретные *projections.FreightRequestsProjection / *SupportTicketsProjection
// удовлетворяют им как есть; в тестах подменяются моками без БД.
type freightLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*projections.FreightRequestListItem, error)
	ListOffers(ctx context.Context, opts ...projections.OfferFilterOption) ([]projections.OfferListItem, error)
}

type supportLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*projections.TicketListItem, error)
}

// Router маппит доменные события из Redis-стримов на получателей в хабе.
// Поля получателей читаются точечным json.Unmarshal нужных полей payload'а —
// без UnmarshalEvent и реестра типов: роутеру не нужна полная структура события.
type Router struct {
	hub     *Hub
	freight freightLookup
	support supportLookup
}

// NewRouter создает роутер событий.
func NewRouter(hub *Hub, freight freightLookup, support supportLookup) *Router {
	return &Router{hub: hub, freight: freight, support: support}
}

// Route доставляет событие подписчикам. Ошибки логируются и не пропагируются:
// SSE — best-effort «пинки», уронить tailer из-за одного события нельзя,
// а пропуск клиент компенсирует refetch'ем при переподключении.
func (r *Router) Route(ctx context.Context, env eventstore.EventEnvelope) {
	// Нет ни одного подключённого клиента — не тратим SQL-lookup'ы на
	// вычисление получателей.
	if r.hub.Len() == 0 {
		return
	}

	switch env.AggregateType {
	case notifevents.AggregateType:
		r.routeNotification(env)
	case frevents.AggregateType:
		r.routeFreightRequest(ctx, env)
	case supportevents.AggregateType:
		r.routeSupportTicket(ctx, env)
	}
}

func (r *Router) routeNotification(env eventstore.EventEnvelope) {
	var p struct {
		MemberID uuid.UUID `json:"member_id"`
	}
	if err := json.Unmarshal(env.Payload, &p); err != nil || p.MemberID == uuid.Nil {
		slog.Debug("sse: notification payload without member_id",
			slog.String("event_type", env.EventType),
			slog.String("aggregate_id", env.AggregateID.String()))
		return
	}

	switch env.EventType {
	case notifevents.TypeInAppCreated:
		r.hub.PublishToMember(p.MemberID, Event{Type: EventNotification, EntityID: env.AggregateID, EventType: env.EventType})
	case notifevents.TypeInAppBatchRead:
		// InAppRead (одиночное) сейчас никем не публикуется — MarkAsRead/
		// MarkAllAsRead шлют только InAppBatchRead; ветку не добавляем, пока
		// не появится продюсер.
		r.hub.PublishToMember(p.MemberID, Event{Type: EventUnread, EntityID: env.AggregateID, EventType: env.EventType})
	}
}

// routeFreightRequest пушит «заявка изменилась» всем причастным организациям:
// заказчику и перевозчикам с офферами. Org id собираются и из payload события,
// и из lookup-проекций — payload спасает, когда проекция ещё не догнала событие
// (например, freight_request.created до UPSERT'а воркером), lookup — когда
// событие не несёт org id (offer.selected содержит только offer_id).
func (r *Router) routeFreightRequest(ctx context.Context, env eventstore.EventEnvelope) {
	orgs := make(map[uuid.UUID]struct{}, 4)

	var p struct {
		CustomerOrgID uuid.UUID `json:"customer_org_id"`
		CarrierOrgID  uuid.UUID `json:"carrier_org_id"`
	}
	if err := json.Unmarshal(env.Payload, &p); err == nil {
		if p.CustomerOrgID != uuid.Nil {
			orgs[p.CustomerOrgID] = struct{}{}
		}
		if p.CarrierOrgID != uuid.Nil {
			orgs[p.CarrierOrgID] = struct{}{}
		}
	}

	if fr, err := r.freight.GetByID(ctx, env.AggregateID); err == nil {
		orgs[fr.CustomerOrgID] = struct{}{}
		if fr.CarrierOrgID != nil {
			orgs[*fr.CarrierOrgID] = struct{}{}
		}
	} else {
		slog.Debug("sse: freight request lookup failed",
			slog.String("aggregate_id", env.AggregateID.String()),
			slog.String("error", err.Error()))
	}

	if offers, err := r.freight.ListOffers(ctx, projections.WithFreightRequestID(env.AggregateID)); err == nil {
		for _, o := range offers {
			orgs[o.CarrierOrgID] = struct{}{}
		}
	} else {
		slog.Debug("sse: offers lookup failed",
			slog.String("aggregate_id", env.AggregateID.String()),
			slog.String("error", err.Error()))
	}

	e := Event{Type: EventFreightRequest, EntityID: env.AggregateID, EventType: env.EventType}
	for orgID := range orgs {
		r.hub.PublishToOrg(orgID, e)
	}
}

// routeSupportTicket пушит владельцу тикета события, инициированные админом
// (плюс reopen — он безвреден и держит вторую вкладку в синхроне). Сообщения
// самого member'а не пушим: его UI перечитывает тикет сразу после отправки.
func (r *Router) routeSupportTicket(ctx context.Context, env eventstore.EventEnvelope) {
	switch env.EventType {
	case supportevents.TypeMessageAdded:
		var p struct {
			SenderType entities.SenderType `json:"sender_type"`
		}
		if err := json.Unmarshal(env.Payload, &p); err != nil || p.SenderType != entities.SenderTypeAdmin {
			return
		}
	case supportevents.TypeTicketClosed, supportevents.TypeTicketReopened:
	default:
		return
	}

	ticket, err := r.support.GetByID(ctx, env.AggregateID)
	if err != nil {
		slog.Debug("sse: support ticket lookup failed",
			slog.String("aggregate_id", env.AggregateID.String()),
			slog.String("error", err.Error()))
		return
	}

	r.hub.PublishToMember(ticket.MemberID, Event{Type: EventSupportTicket, EntityID: env.AggregateID, EventType: env.EventType})
}
