package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	notifApp "codeberg.org/udison/veziizi/backend/internal/application/notification"
	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	orderEvents "codeberg.org/udison/veziizi/backend/internal/domain/order/events"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

// NotificationDispatcherHandler маппит domain events в уведомления
type NotificationDispatcherHandler struct {
	notifService     *notifApp.Service
	frProjection     *projections.FreightRequestsProjection
	ordersProjection *projections.OrdersProjection
	membersProjection *projections.MembersProjection
	publisher        message.Publisher
}

// NewNotificationDispatcherHandler создает новый handler
func NewNotificationDispatcherHandler(
	notifService *notifApp.Service,
	frProjection *projections.FreightRequestsProjection,
	ordersProjection *projections.OrdersProjection,
	membersProjection *projections.MembersProjection,
	publisher message.Publisher,
) *NotificationDispatcherHandler {
	return &NotificationDispatcherHandler{
		notifService:      notifService,
		frProjection:      frProjection,
		ordersProjection:  ordersProjection,
		membersProjection: membersProjection,
		publisher:         publisher,
	}
}

// TelegramNotification сообщение для telegram-sender
type TelegramNotification struct {
	MemberID uuid.UUID `json:"member_id"`
	ChatID   int64     `json:"chat_id"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	Link     string    `json:"link,omitempty"`
}

// Handle обрабатывает событие
func (h *NotificationDispatcherHandler) Handle(msg *message.Message) error {
	var envelope eventstore.EventEnvelope
	if err := json.Unmarshal(msg.Payload, &envelope); err != nil {
		slog.Error("failed to unmarshal event envelope", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	evt, err := envelope.UnmarshalEvent()
	if err != nil {
		slog.Error("failed to unmarshal event", slog.String("error", err.Error()))
		return fmt.Errorf("unmarshal event: %w", err)
	}

	return h.handleEvent(msg.Context(), evt)
}

func (h *NotificationDispatcherHandler) handleEvent(ctx context.Context, evt eventstore.Event) error {
	switch e := evt.(type) {
	// FreightRequest events
	case frEvents.FreightRequestCreated:
		return h.onFreightRequestCreated(ctx, e)
	case frEvents.OfferMade:
		return h.onOfferMade(ctx, e)
	case frEvents.OfferSelected:
		return h.onOfferSelected(ctx, e)
	case frEvents.OfferRejected:
		return h.onOfferRejected(ctx, e)
	case frEvents.OfferConfirmed:
		return h.onOfferConfirmed(ctx, e)
	case frEvents.OfferDeclined:
		return h.onOfferDeclined(ctx, e)
	case frEvents.OfferWithdrawn:
		return h.onOfferWithdrawn(ctx, e)

	// Order events
	case orderEvents.OrderCreated:
		return h.onOrderCreated(ctx, e)
	case orderEvents.MessageSent:
		return h.onMessageSent(ctx, e)
	case orderEvents.OrderCompleted:
		return h.onOrderCompleted(ctx, e)
	case orderEvents.OrderCancelled:
		return h.onOrderCancelled(ctx, e)
	}

	return nil
}

// ===============================
// FreightRequest Events
// ===============================

func (h *NotificationDispatcherHandler) onFreightRequestCreated(ctx context.Context, e frEvents.FreightRequestCreated) error {
	// Отправляем уведомление всем активным членам (кроме создателя)
	memberIDs, err := h.membersProjection.GetAllActiveMemberIDs(ctx, &e.CustomerMemberID)
	if err != nil {
		return fmt.Errorf("get active member IDs: %w", err)
	}

	slog.Info("sending new freight request notifications",
		slog.String("freight_request_id", e.AggregateID().String()),
		slog.Int("recipient_count", len(memberIDs)))

	title := "Новая заявка"

	// Формируем текст маршрута из первой и последней точки
	routeText := ""
	if len(e.Route.Points) >= 2 {
		from := e.Route.Points[0].Address
		to := e.Route.Points[len(e.Route.Points)-1].Address
		routeText = fmt.Sprintf(": %s → %s", from, to)
	}
	body := fmt.Sprintf("Заявка #%d%s", e.RequestNumber, routeText)
	link := fmt.Sprintf("/freight-requests/%s", e.AggregateID())

	for _, memberID := range memberIDs {
		// Получаем org_id для member (нужен для notification)
		member, err := h.membersProjection.GetByID(ctx, memberID)
		if err != nil || member == nil {
			continue
		}

		if err := h.createNotification(ctx, notificationInput{
			MemberID:         memberID,
			OrganizationID:   member.OrganizationID,
			NotificationType: values.TypeNewFreightRequest,
			Title:            title,
			Body:             body,
			Link:             link,
			EntityType:       values.EntityFreightRequest,
			EntityID:         e.AggregateID(),
		}); err != nil {
			slog.Warn("failed to send freight request notification",
				slog.String("member_id", memberID.String()),
				slog.String("error", err.Error()))
		}
	}

	return nil
}

func (h *NotificationDispatcherHandler) onOfferMade(ctx context.Context, e frEvents.OfferMade) error {
	// Уведомляем заказчика о новом предложении
	fr, err := h.frProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		// Projection ещё не обновилась — пропускаем, не блокируем очередь
		slog.Warn("freight request not found in projection, skipping notification",
			slog.String("freight_request_id", e.AggregateID().String()))
		return nil
	}
	if fr == nil || fr.CustomerMemberID == nil {
		return nil
	}

	return h.createNotification(ctx, notificationInput{
		MemberID:         *fr.CustomerMemberID,
		OrganizationID:   fr.CustomerOrgID,
		NotificationType: values.TypeNewOffer,
		Title:            "Новое предложение",
		Body:             fmt.Sprintf("На заявку #%d поступило новое предложение", fr.RequestNumber),
		Link:             fmt.Sprintf("/freight-requests/%s", fr.ID),
		EntityType:       values.EntityFreightRequest,
		EntityID:         fr.ID,
	})
}

func (h *NotificationDispatcherHandler) onOfferSelected(ctx context.Context, e frEvents.OfferSelected) error {
	// Уведомляем перевозчика что его предложение выбрано
	offer, err := h.frProjection.GetOfferByID(ctx, e.OfferID)
	if err != nil {
		return fmt.Errorf("get offer: %w", err)
	}
	if offer == nil {
		return nil
	}

	fr, err := h.frProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		slog.Warn("freight request not found in projection, skipping notification",
			slog.String("freight_request_id", e.AggregateID().String()))
		return nil
	}
	if fr == nil {
		return nil
	}

	// Нужен carrier_member_id - но его нет в offers_lookup
	// Придётся добавить позже или использовать события
	// Пока пропускаем - этот member_id есть в событии OfferMade
	slog.Info("offer selected notification skipped - carrier member lookup not implemented",
		slog.String("offer_id", e.OfferID.String()))

	return nil
}

func (h *NotificationDispatcherHandler) onOfferRejected(ctx context.Context, e frEvents.OfferRejected) error {
	// Аналогично onOfferSelected - нужен carrier_member_id
	slog.Info("offer rejected notification skipped - carrier member lookup not implemented",
		slog.String("offer_id", e.OfferID.String()))
	return nil
}

func (h *NotificationDispatcherHandler) onOfferConfirmed(ctx context.Context, e frEvents.OfferConfirmed) error {
	// Уведомляем заказчика что предложение подтверждено
	fr, err := h.frProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		slog.Warn("freight request not found in projection, skipping notification",
			slog.String("freight_request_id", e.AggregateID().String()))
		return nil
	}
	if fr == nil || fr.CustomerMemberID == nil {
		return nil
	}

	return h.createNotification(ctx, notificationInput{
		MemberID:         *fr.CustomerMemberID,
		OrganizationID:   fr.CustomerOrgID,
		NotificationType: values.TypeOfferConfirmed,
		Title:            "Предложение подтверждено",
		Body:             fmt.Sprintf("По заявке #%d создан заказ", fr.RequestNumber),
		Link:             fmt.Sprintf("/freight-requests/%s", fr.ID),
		EntityType:       values.EntityFreightRequest,
		EntityID:         fr.ID,
	})
}

func (h *NotificationDispatcherHandler) onOfferDeclined(ctx context.Context, e frEvents.OfferDeclined) error {
	// Уведомляем заказчика что перевозчик отклонил
	fr, err := h.frProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		slog.Warn("freight request not found in projection, skipping notification",
			slog.String("freight_request_id", e.AggregateID().String()))
		return nil
	}
	if fr == nil || fr.CustomerMemberID == nil {
		return nil
	}

	return h.createNotification(ctx, notificationInput{
		MemberID:         *fr.CustomerMemberID,
		OrganizationID:   fr.CustomerOrgID,
		NotificationType: values.TypeOfferDeclined,
		Title:            "Предложение отклонено",
		Body:             fmt.Sprintf("Перевозчик отклонил выбранное предложение по заявке #%d", fr.RequestNumber),
		Link:             fmt.Sprintf("/freight-requests/%s", fr.ID),
		EntityType:       values.EntityFreightRequest,
		EntityID:         fr.ID,
	})
}

func (h *NotificationDispatcherHandler) onOfferWithdrawn(ctx context.Context, e frEvents.OfferWithdrawn) error {
	// Уведомляем заказчика что предложение отозвано
	fr, err := h.frProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		slog.Warn("freight request not found in projection, skipping notification",
			slog.String("freight_request_id", e.AggregateID().String()))
		return nil
	}
	if fr == nil || fr.CustomerMemberID == nil {
		return nil
	}

	return h.createNotification(ctx, notificationInput{
		MemberID:         *fr.CustomerMemberID,
		OrganizationID:   fr.CustomerOrgID,
		NotificationType: values.TypeOfferWithdrawn,
		Title:            "Предложение отозвано",
		Body:             fmt.Sprintf("Перевозчик отозвал своё предложение по заявке #%d", fr.RequestNumber),
		Link:             fmt.Sprintf("/freight-requests/%s", fr.ID),
		EntityType:       values.EntityFreightRequest,
		EntityID:         fr.ID,
	})
}

// ===============================
// Order Events
// ===============================

func (h *NotificationDispatcherHandler) onOrderCreated(ctx context.Context, e orderEvents.OrderCreated) error {
	// Уведомляем обе стороны о создании заказа
	errors := make([]error, 0)

	// Заказчику
	if err := h.createNotification(ctx, notificationInput{
		MemberID:         e.CustomerMemberID,
		OrganizationID:   e.CustomerOrgID,
		NotificationType: values.TypeOrderCreated,
		Title:            "Заказ создан",
		Body:             fmt.Sprintf("Создан заказ #%d", e.OrderNumber),
		Link:             fmt.Sprintf("/orders/%s", e.AggregateID()),
		EntityType:       values.EntityOrder,
		EntityID:         e.AggregateID(),
	}); err != nil {
		errors = append(errors, fmt.Errorf("customer: %w", err))
	}

	// Перевозчику
	if err := h.createNotification(ctx, notificationInput{
		MemberID:         e.CarrierMemberID,
		OrganizationID:   e.CarrierOrgID,
		NotificationType: values.TypeOrderCreated,
		Title:            "Заказ создан",
		Body:             fmt.Sprintf("Создан заказ #%d", e.OrderNumber),
		Link:             fmt.Sprintf("/orders/%s", e.AggregateID()),
		EntityType:       values.EntityOrder,
		EntityID:         e.AggregateID(),
	}); err != nil {
		errors = append(errors, fmt.Errorf("carrier: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("order created notifications: %v", errors)
	}
	return nil
}

func (h *NotificationDispatcherHandler) onMessageSent(ctx context.Context, e orderEvents.MessageSent) error {
	// Уведомляем другую сторону о новом сообщении
	order, err := h.ordersProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		slog.Warn("order not found in projection, skipping notification",
			slog.String("order_id", e.AggregateID().String()))
		return nil
	}
	if order == nil {
		return nil
	}

	// Определяем получателя (другая сторона)
	var recipientMemberID uuid.UUID
	var recipientOrgID uuid.UUID

	if e.SenderOrgID == order.CustomerOrgID {
		recipientMemberID = order.CarrierMemberID
		recipientOrgID = order.CarrierOrgID
	} else {
		recipientMemberID = order.CustomerMemberID
		recipientOrgID = order.CustomerOrgID
	}

	return h.createNotification(ctx, notificationInput{
		MemberID:         recipientMemberID,
		OrganizationID:   recipientOrgID,
		NotificationType: values.TypeOrderMessage,
		Title:            "Новое сообщение",
		Body:             fmt.Sprintf("В заказе #%d новое сообщение", order.OrderNumber),
		Link:             fmt.Sprintf("/orders/%s", order.ID),
		EntityType:       values.EntityOrder,
		EntityID:         order.ID,
	})
}

func (h *NotificationDispatcherHandler) onOrderCompleted(ctx context.Context, e orderEvents.OrderCompleted) error {
	// Уведомляем обе стороны о завершении
	order, err := h.ordersProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		slog.Warn("order not found in projection, skipping notification",
			slog.String("order_id", e.AggregateID().String()))
		return nil
	}
	if order == nil {
		return nil
	}

	errors := make([]error, 0)

	// Заказчику
	if err := h.createNotification(ctx, notificationInput{
		MemberID:         order.CustomerMemberID,
		OrganizationID:   order.CustomerOrgID,
		NotificationType: values.TypeOrderCompleted,
		Title:            "Заказ завершён",
		Body:             fmt.Sprintf("Заказ #%d успешно завершён", order.OrderNumber),
		Link:             fmt.Sprintf("/orders/%s", order.ID),
		EntityType:       values.EntityOrder,
		EntityID:         order.ID,
	}); err != nil {
		errors = append(errors, fmt.Errorf("customer: %w", err))
	}

	// Перевозчику
	if err := h.createNotification(ctx, notificationInput{
		MemberID:         order.CarrierMemberID,
		OrganizationID:   order.CarrierOrgID,
		NotificationType: values.TypeOrderCompleted,
		Title:            "Заказ завершён",
		Body:             fmt.Sprintf("Заказ #%d успешно завершён", order.OrderNumber),
		Link:             fmt.Sprintf("/orders/%s", order.ID),
		EntityType:       values.EntityOrder,
		EntityID:         order.ID,
	}); err != nil {
		errors = append(errors, fmt.Errorf("carrier: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("order completed notifications: %v", errors)
	}
	return nil
}

func (h *NotificationDispatcherHandler) onOrderCancelled(ctx context.Context, e orderEvents.OrderCancelled) error {
	// Уведомляем другую сторону об отмене
	order, err := h.ordersProjection.GetByID(ctx, e.AggregateID())
	if err != nil {
		slog.Warn("order not found in projection, skipping notification",
			slog.String("order_id", e.AggregateID().String()))
		return nil
	}
	if order == nil {
		return nil
	}

	// Определяем получателя (другая сторона)
	var recipientMemberID uuid.UUID
	var recipientOrgID uuid.UUID

	if e.CancelledByOrgID == order.CustomerOrgID {
		recipientMemberID = order.CarrierMemberID
		recipientOrgID = order.CarrierOrgID
	} else {
		recipientMemberID = order.CustomerMemberID
		recipientOrgID = order.CustomerOrgID
	}

	return h.createNotification(ctx, notificationInput{
		MemberID:         recipientMemberID,
		OrganizationID:   recipientOrgID,
		NotificationType: values.TypeOrderCancelled,
		Title:            "Заказ отменён",
		Body:             fmt.Sprintf("Заказ #%d был отменён", order.OrderNumber),
		Link:             fmt.Sprintf("/orders/%s", order.ID),
		EntityType:       values.EntityOrder,
		EntityID:         order.ID,
	})
}

// ===============================
// Helper
// ===============================

type notificationInput struct {
	MemberID         uuid.UUID
	OrganizationID   uuid.UUID
	NotificationType values.NotificationType
	Title            string
	Body             string
	Link             string
	EntityType       values.EntityType
	EntityID         uuid.UUID
}

func (h *NotificationDispatcherHandler) createNotification(ctx context.Context, input notificationInput) error {
	// Проверяем настройки для in_app
	shouldInApp, err := h.notifService.ShouldNotify(ctx, input.MemberID, input.NotificationType, values.ChannelInApp)
	if err != nil {
		slog.Warn("failed to check in_app settings, using default",
			slog.String("member_id", input.MemberID.String()),
			slog.String("error", err.Error()))
		shouldInApp = true // default
	}

	if shouldInApp {
		if err := h.notifService.CreateInApp(ctx, notifApp.CreateNotificationInput{
			MemberID:         input.MemberID,
			OrganizationID:   input.OrganizationID,
			NotificationType: input.NotificationType,
			Title:            input.Title,
			Body:             input.Body,
			Link:             input.Link,
			EntityType:       input.EntityType,
			EntityID:         input.EntityID,
		}); err != nil {
			slog.Error("failed to create in-app notification",
				slog.String("member_id", input.MemberID.String()),
				slog.String("type", string(input.NotificationType)),
				slog.String("error", err.Error()))
		} else {
			slog.Info("in-app notification created",
				slog.String("member_id", input.MemberID.String()),
				slog.String("type", string(input.NotificationType)))
		}
	}

	// Проверяем настройки для telegram
	shouldTelegram, err := h.notifService.ShouldNotify(ctx, input.MemberID, input.NotificationType, values.ChannelTelegram)
	if err != nil {
		slog.Warn("failed to check telegram settings",
			slog.String("member_id", input.MemberID.String()),
			slog.String("error", err.Error()))
		shouldTelegram = false
	}

	if shouldTelegram {
		chatID, err := h.notifService.GetTelegramChatID(ctx, input.MemberID)
		if err != nil || chatID == nil {
			slog.Warn("telegram enabled but no chat_id",
				slog.String("member_id", input.MemberID.String()))
			return nil
		}

		telegramMsg := TelegramNotification{
			MemberID: input.MemberID,
			ChatID:   *chatID,
			Title:    input.Title,
			Body:     input.Body,
			Link:     input.Link,
		}

		payload, err := json.Marshal(telegramMsg)
		if err != nil {
			return fmt.Errorf("marshal telegram notification: %w", err)
		}

		msg := message.NewMessage(uuid.New().String(), payload)
		if err := h.publisher.Publish("notification.telegram", msg); err != nil {
			slog.Error("failed to publish telegram notification",
				slog.String("member_id", input.MemberID.String()),
				slog.String("error", err.Error()))
		} else {
			slog.Info("telegram notification published",
				slog.String("member_id", input.MemberID.String()),
				slog.String("type", string(input.NotificationType)))
		}
	}

	return nil
}
