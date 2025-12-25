package freightrequest

import (
	"context"
	"fmt"
	"strings"

	frEvents "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/events"
	frValues "codeberg.org/udison/veziizi/backend/internal/domain/freightrequest/values"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/rules"
	"codeberg.org/udison/veziizi/backend/internal/domain/notification/values"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// Русские названия типов груза
var cargoTypeLabels = map[frValues.CargoType]string{
	frValues.CargoTypeGeneral:      "Генеральный",
	frValues.CargoTypeBulk:         "Насыпной",
	frValues.CargoTypeLiquid:       "Наливной",
	frValues.CargoTypeRefrigerated: "Рефрижераторный",
	frValues.CargoTypeDangerous:    "Опасный",
	frValues.CargoTypeOversized:    "Негабаритный",
	frValues.CargoTypeContainer:    "Контейнерный",
}

// Русские названия типов кузова
var bodyTypeLabels = map[frValues.BodyType]string{
	frValues.BodyTypeTent:         "Тент",
	frValues.BodyTypeRefrigerator: "Рефрижератор",
	frValues.BodyTypeIsothermal:   "Изотерм",
	frValues.BodyTypeContainer:    "Контейнеровоз",
	frValues.BodyTypeOpenbed:      "Открытая",
	frValues.BodyTypeLowbed:       "Низкорамник",
	frValues.BodyTypeJumbo:        "Джамбо",
	frValues.BodyTypeTank:         "Цистерна",
	frValues.BodyTypeTipper:       "Самосвал",
}

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
		routeText = fmt.Sprintf("%s → %s", from, to)
	}

	// Формируем подробное описание
	title := fmt.Sprintf("Заявка #%d", e.RequestNumber)

	var bodyParts []string

	// Маршрут
	if routeText != "" {
		bodyParts = append(bodyParts, fmt.Sprintf("📍 %s", routeText))
	}

	// Груз: тип и вес
	cargoLabel := cargoTypeLabels[e.Cargo.Type]
	if cargoLabel == "" {
		cargoLabel = e.Cargo.Type.String()
	}
	bodyParts = append(bodyParts, fmt.Sprintf("📦 %s, %.1f т", cargoLabel, e.Cargo.Weight))

	// Кузов
	if len(e.VehicleRequirements.BodyTypes) > 0 {
		var bodyNames []string
		for _, bt := range e.VehicleRequirements.BodyTypes {
			label := bodyTypeLabels[bt]
			if label == "" {
				label = bt.String()
			}
			bodyNames = append(bodyNames, label)
		}
		bodyParts = append(bodyParts, fmt.Sprintf("🚛 %s", strings.Join(bodyNames, ", ")))
	}

	// Цена
	if e.Payment.Price != nil {
		priceRub := float64(e.Payment.Price.Amount) / 100
		bodyParts = append(bodyParts, fmt.Sprintf("💰 %.0f %s", priceRub, e.Payment.Price.Currency.String()))
	}

	body := strings.Join(bodyParts, "\n")
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
