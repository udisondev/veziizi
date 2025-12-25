package rules

import (
	"context"
	"fmt"
	"log/slog"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// Registry хранит и выполняет правила уведомлений
type Registry struct {
	rules map[string][]NotificationRule
}

// NewRegistry создает новый реестр правил
func NewRegistry() *Registry {
	return &Registry{
		rules: make(map[string][]NotificationRule),
	}
}

// Register регистрирует правило для типа события
func (r *Registry) Register(rule NotificationRule) {
	eventType := rule.EventType()
	r.rules[eventType] = append(r.rules[eventType], rule)
	slog.Debug("registered notification rule",
		slog.String("event_type", eventType),
		slog.String("rule", fmt.Sprintf("%T", rule)))
}

// Process обрабатывает событие через все зарегистрированные правила
// Возвращает все запросы на уведомления от всех правил
func (r *Registry) Process(ctx context.Context, event eventstore.Event) ([]NotificationRequest, error) {
	eventType := event.EventType()
	rules, ok := r.rules[eventType]
	if !ok {
		return nil, nil // Нет правил для этого события
	}

	var requests []NotificationRequest
	for _, rule := range rules {
		reqs, err := rule.Process(ctx, event)
		if err != nil {
			slog.Error("notification rule failed",
				slog.String("event_type", eventType),
				slog.String("rule", fmt.Sprintf("%T", rule)),
				slog.String("error", err.Error()))
			// Продолжаем обработку других правил - не блокируем очередь
			continue
		}
		requests = append(requests, reqs...)
	}

	return requests, nil
}

// HasRules проверяет есть ли правила для типа события
func (r *Registry) HasRules(eventType string) bool {
	_, ok := r.rules[eventType]
	return ok
}
