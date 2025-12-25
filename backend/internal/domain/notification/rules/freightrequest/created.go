package freightrequest

import (
	"context"
	"fmt"

	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// FreightRequestCreatedRule уведомляет подписчиков о новой заявке
type FreightRequestCreatedRule struct {
	deps               rules.Dependencies
	subscriberResolver rules.SubscribedMembersResolver
}

// NewFreightRequestCreatedRule создает правило
// Если subscriberResolver nil - правило пропускает уведомления
func NewFreightRequestCreatedRule(deps rules.Dependencies, subscriberResolver rules.SubscribedMembersResolver) *FreightRequestCreatedRule {
	return &FreightRequestCreatedRule{
		deps:               deps,
		subscriberResolver: subscriberResolver,
	}
}

func (r *FreightRequestCreatedRule) EventType() string {
	return frEvents.TypeFreightRequestCreated
}

func (r *FreightRequestCreatedRule) Process(ctx context.Context, event eventstore.Event) ([]rules.NotificationRequest, error) {
	e, ok := event.(frEvents.FreightRequestCreated)
	if !ok {
		return nil, fmt.Errorf("unexpected event type: %T", event)
	}

	// Если resolver не настроен - пропускаем
	if r.subscriberResolver == nil {
		return nil, nil
	}

	// Формируем фильтр из данных заявки
	filter := rules.SubscriptionFilter{}
	if len(e.Route.Points) >= 2 {
		filter.OriginCountryID = e.Route.Points[0].CountryID
		filter.DestinationCountryID = e.Route.Points[len(e.Route.Points)-1].CountryID
	}
	filter.CargoType = e.Cargo.Type.String()
	filter.CargoWeight = e.Cargo.Weight

	// Получаем подписчиков
	subscribers, err := r.subscriberResolver.GetSubscribedMembers(ctx, filter, e.CustomerMemberID)
	if err != nil {
		return nil, fmt.Errorf("get subscribed members: %w", err)
	}

	// Формируем текст маршрута
	routeText := ""
	if len(e.Route.Points) >= 2 {
		from := e.Route.Points[0].Address
		to := e.Route.Points[len(e.Route.Points)-1].Address
		routeText = fmt.Sprintf(": %s → %s", from, to)
	}

	title := "Новая заявка"
	body := fmt.Sprintf("Заявка #%d%s", e.RequestNumber, routeText)
	link := fmt.Sprintf("/freight-requests/%s", e.AggregateID())

	requests := make([]rules.NotificationRequest, 0, len(subscribers))
	for _, sub := range subscribers {
		requests = append(requests, rules.NotificationRequest{
			RecipientMemberID: sub.MemberID,
			RecipientOrgID:    sub.OrganizationID,
			NotificationType:  values.TypeNewFreightRequest,
			Title:             title,
			Body:              body,
			Link:              link,
			EntityType:        values.EntityFreightRequest,
			EntityID:          e.AggregateID(),
		})
	}

	return requests, nil
}
