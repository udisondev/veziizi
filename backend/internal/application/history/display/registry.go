package display

import (
	"context"
	"fmt"
	"strings"

	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/projections"
)

// Registry хранит форматтеры и предоставляет единый интерфейс для форматирования событий
type Registry struct {
	formatters    []EventFormatter
	members       *projections.MembersProjection
	organizations *projections.OrganizationsProjection
}

// NewRegistry создаёт новый Registry с проекциями для резолва
func NewRegistry(
	members *projections.MembersProjection,
	organizations *projections.OrganizationsProjection,
) *Registry {
	r := &Registry{
		members:       members,
		organizations: organizations,
	}

	// Регистрируем форматтеры
	r.formatters = []EventFormatter{
		NewOrganizationFormatter(),
		NewFreightRequestFormatter(),
		NewReviewFormatter(),
	}

	return r
}

// Format форматирует событие, используя подходящий форматтер
func (r *Registry) Format(ctx context.Context, event eventstore.Event) (DisplayView, error) {
	eventType := event.EventType()

	// Ищем подходящий форматтер
	for _, f := range r.formatters {
		if f.Supports(eventType) {
			resolver := NewCachedResolver(r.members, r.organizations)
			return f.Format(ctx, event, resolver)
		}
	}

	// Fallback для неизвестных событий
	return r.fallbackFormat(eventType), nil
}

// FormatWithResolver форматирует событие с переданным resolver (для batch)
func (r *Registry) FormatWithResolver(ctx context.Context, event eventstore.Event, resolver EntityResolver) (DisplayView, error) {
	eventType := event.EventType()

	for _, f := range r.formatters {
		if f.Supports(eventType) {
			return f.Format(ctx, event, resolver)
		}
	}

	return r.fallbackFormat(eventType), nil
}

// NewResolver создаёт новый CachedResolver для batch операций
func (r *Registry) NewResolver() *CachedResolver {
	return NewCachedResolver(r.members, r.organizations)
}

// fallbackFormat создаёт базовое отображение для неизвестных событий
func (r *Registry) fallbackFormat(eventType string) DisplayView {
	// Преобразуем event_type в человекочитаемый заголовок
	parts := strings.Split(eventType, ".")
	title := "Событие"
	if len(parts) > 0 {
		title = humanizeEventType(eventType)
	}

	return DisplayView{
		Title:       title,
		Description: fmt.Sprintf("Событие типа %s", eventType),
		Severity:    "info",
	}
}

// humanizeEventType преобразует event type в читаемый заголовок
func humanizeEventType(eventType string) string {
	// Маппинг для известных типов
	labels := map[string]string{
		"organization.created":   "Организация создана",
		"organization.approved":  "Организация одобрена",
		"organization.rejected":  "Организация отклонена",
		"organization.suspended": "Организация приостановлена",
		"organization.updated":   "Организация обновлена",

		"member.added":        "Сотрудник добавлен",
		"member.removed":      "Сотрудник удалён",
		"member.role_changed": "Роль изменена",
		"member.blocked":      "Сотрудник заблокирован",
		"member.unblocked":    "Сотрудник разблокирован",

		"invitation.created":   "Приглашение создано",
		"invitation.accepted":  "Приглашение принято",
		"invitation.cancelled": "Приглашение отменено",
		"invitation.expired":   "Приглашение истекло",

		"fraudster.marked":   "Отмечен как мошенник",
		"fraudster.unmarked": "Снята метка мошенника",

		"freight_request.created":                   "Заявка создана",
		"freight_request.updated":                   "Заявка обновлена",
		"freight_request.reassigned":                "Заявка переназначена",
		"freight_request.cancelled":                 "Заявка отменена",
		"freight_request.expired":                   "Заявка истекла",
		"freight_request.customer_completed":        "Заказчик завершил перевозку",
		"freight_request.carrier_completed":         "Перевозчик завершил перевозку",
		"freight_request.completed":                 "Перевозка завершена",
		"freight_request.review_left":               "Отзыв оставлен",
		"freight_request.cancelled_after_confirmed": "Отменено после подтверждения",
		"freight_request.carrier_member_reassigned": "Ответственный перевозчика изменён",

		"offer.made":      "Оффер сделан",
		"offer.withdrawn": "Оффер отозван",
		"offer.selected":  "Оффер выбран",
		"offer.rejected":  "Оффер отклонён",
		"offer.confirmed": "Оффер подтверждён",
		"offer.declined":  "Оффер отклонён перевозчиком",

		"review.received":    "Отзыв получен",
		"review.analyzed":    "Отзыв проанализирован",
		"review.approved":    "Отзыв одобрен",
		"review.rejected":    "Отзыв отклонён",
		"review.activated":   "Отзыв активирован",
		"review.deactivated": "Отзыв деактивирован",
	}

	if label, ok := labels[eventType]; ok {
		return label
	}

	return eventType
}
