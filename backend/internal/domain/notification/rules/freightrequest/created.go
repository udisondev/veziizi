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

// Русские названия типов транспорта
var vehicleTypeLabels = map[frValues.VehicleType]string{
	frValues.VehicleTypeVan:              "Фургон",
	frValues.VehicleTypeFlatbed:          "Платформа",
	frValues.VehicleTypeTanker:           "Цистерна",
	frValues.VehicleTypeDumpTruck:        "Самосвал",
	frValues.VehicleTypeSpecializedTruck: "Спецтранспорт",
	frValues.VehicleTypeLightTruck:       "Легкий грузовик",
	frValues.VehicleTypeMediumTruck:      "Средний грузовик",
	frValues.VehicleTypeHeavyTruck:       "Тяжелый грузовик",
}

// FreightRequestCreatedRule уведомляет подписчиков о новой заявке
type FreightRequestCreatedRule struct {
	deps                rules.Dependencies
	subscriptionMatcher rules.SubscriptionMatcher
}

// NewFreightRequestCreatedRule создает правило
// Если subscriptionMatcher nil - правило пропускает уведомления
func NewFreightRequestCreatedRule(deps rules.Dependencies, subscriptionMatcher rules.SubscriptionMatcher) *FreightRequestCreatedRule {
	return &FreightRequestCreatedRule{
		deps:                deps,
		subscriptionMatcher: subscriptionMatcher,
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

	// Если matcher не настроен - пропускаем
	if r.subscriptionMatcher == nil {
		return nil, nil
	}

	// Формируем данные заявки для matching
	matchData := frValues.FreightRequestMatchData{
		CustomerMemberID: e.CustomerMemberID,
		Route:            e.Route,
		Cargo:            e.Cargo,
		Payment:          e.Payment,
		VehicleReqs:      e.VehicleRequirements,
	}

	// Находим подходящие подписки
	matches, err := r.subscriptionMatcher.FindMatchingSubscriptions(ctx, matchData, e.CustomerMemberID)
	if err != nil {
		return nil, fmt.Errorf("find matching subscriptions: %w", err)
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// Формируем текст маршрута
	routeText := ""
	if len(e.Route.Points) >= 2 {
		from := e.Route.Points[0].Address
		to := e.Route.Points[len(e.Route.Points)-1].Address
		routeText = fmt.Sprintf("%s → %s", from, to)
	}

	// Формируем body для уведомления
	var bodyParts []string
	if routeText != "" {
		bodyParts = append(bodyParts, fmt.Sprintf("📍 %s", routeText))
	}
	bodyParts = append(bodyParts, fmt.Sprintf("📦 %.1f т", e.Cargo.Weight))

	// Транспорт
	vehicleLabel := vehicleTypeLabels[e.VehicleRequirements.VehicleType]
	if vehicleLabel == "" {
		vehicleLabel = e.VehicleRequirements.VehicleType.String()
	}
	bodyParts = append(bodyParts, fmt.Sprintf("🚛 %s", vehicleLabel))

	// Цена
	if e.Payment.Price != nil {
		priceRub := float64(e.Payment.Price.Amount) / 100
		bodyParts = append(bodyParts, fmt.Sprintf("💰 %.0f %s", priceRub, e.Payment.Price.Currency.String()))
	}

	body := strings.Join(bodyParts, "\n")
	link := fmt.Sprintf("/freight-requests/%s", e.AggregateID())

	// Формируем уведомления для каждой подходящей подписки
	requests := make([]rules.NotificationRequest, 0, len(matches))
	for _, match := range matches {
		// Добавляем название подписки в заголовок
		title := fmt.Sprintf("Заявка #%d (%s)", e.RequestNumber, match.SubscriptionName)

		requests = append(requests, rules.NotificationRequest{
			RecipientMemberID: match.MemberID,
			RecipientOrgID:    match.OrganizationID,
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
