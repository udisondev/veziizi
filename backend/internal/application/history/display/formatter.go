package display

import (
	"context"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
)

// EventFormatter форматирует событие в человекочитаемый вид
type EventFormatter interface {
	// Supports проверяет, поддерживает ли форматтер данный тип события
	Supports(eventType string) bool

	// Format форматирует событие в DisplayView
	Format(ctx context.Context, event eventstore.Event, resolver EntityResolver) (DisplayView, error)
}
