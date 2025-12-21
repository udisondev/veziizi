package history

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/application/history/display"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/infrastructure/projections"
	"github.com/google/uuid"
)

// ActorInfo represents information about the user who initiated an event
type ActorInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

// EventHistoryItem represents a single event in the history
type EventHistoryItem struct {
	ID         uuid.UUID      `json:"id"`
	EventType  string         `json:"event_type"`
	Version    int64          `json:"version"`
	OccurredAt time.Time      `json:"occurred_at"`
	Actor      *ActorInfo     `json:"actor,omitempty"`
	Data       map[string]any `json:"data"`
}

// EventHistoryPage represents a paginated list of events
type EventHistoryPage struct {
	Items []EventHistoryItem `json:"items"`
	Total int                `json:"total"`
}

// DisplayableHistoryItem extends EventHistoryItem with human-readable display
type DisplayableHistoryItem struct {
	ID         uuid.UUID       `json:"id"`
	EventType  string          `json:"event_type"`
	Version    int64           `json:"version"`
	OccurredAt time.Time       `json:"occurred_at"`
	Actor      *ActorInfo      `json:"actor,omitempty"`
	Display    display.DisplayView `json:"display"`
}

// DisplayableHistoryPage represents a paginated list of displayable events
type DisplayableHistoryPage struct {
	Items []DisplayableHistoryItem `json:"items"`
	Total int                      `json:"total"`
}

// Service provides history functionality for aggregates
type Service struct {
	eventStore        eventstore.Store
	membersProjection *projections.MembersProjection
	displayRegistry   *display.Registry
}

// NewService creates a new history service
func NewService(
	eventStore eventstore.Store,
	membersProjection *projections.MembersProjection,
	displayRegistry *display.Registry,
) *Service {
	return &Service{
		eventStore:        eventStore,
		membersProjection: membersProjection,
		displayRegistry:   displayRegistry,
	}
}

// actorFieldMapping maps event types to their actor field names
var actorFieldMapping = map[string]string{
	// Organization events
	"organization.approved":  "approved_by",
	"organization.rejected":  "rejected_by",
	"organization.suspended": "suspended_by",
	"organization.updated":   "updated_by",

	// Member events
	"member.added":        "invited_by",
	"member.role_changed": "changed_by",
	"member.blocked":      "blocked_by",
	"member.unblocked":    "unblocked_by",

	// Invitation events
	"invitation.created":   "created_by",
	"invitation.cancelled": "cancelled_by",

	// FreightRequest events
	"freight_request.created":    "customer_member_id",
	"freight_request.updated":    "updated_by",
	"freight_request.reassigned": "reassigned_by",
	"freight_request.cancelled":  "cancelled_by",

	// Offer events
	"offer.made":      "carrier_member_id",
	"offer.withdrawn": "withdrawn_by",
	"offer.selected":  "selected_by",
	"offer.rejected":  "rejected_by",
	"offer.confirmed": "confirmed_by",
	"offer.declined":  "declined_by",

	// Order events
	"order.cancelled":          "cancelled_by_member_id",
	"order.customer_completed": "member_id",
	"order.carrier_completed":  "member_id",
	"order.message_sent":       "sender_member_id",
	"order.document_attached":  "uploaded_by",
	"order.document_removed":   "removed_by",
}

// adminEventTypes are events where the actor is a platform admin, not a member
var adminEventTypes = map[string]bool{
	"organization.approved":  true,
	"organization.rejected":  true,
	"organization.suspended": true,
}

// sensitiveFields are fields that should be removed from event data
var sensitiveFields = map[string]bool{
	"password_hash": true,
	"token":         true,
}

// GetHistory retrieves paginated event history for an aggregate
func (s *Service) GetHistory(ctx context.Context, aggregateID uuid.UUID, aggregateType string, limit, offset int) (*EventHistoryPage, error) {
	envelopes, total, err := s.eventStore.LoadPaginated(ctx, aggregateID, aggregateType, limit, offset)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			return &EventHistoryPage{Items: []EventHistoryItem{}, Total: 0}, nil
		}
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	items := make([]EventHistoryItem, 0, len(envelopes))
	for _, env := range envelopes {
		item, err := s.envelopeToHistoryItem(ctx, env)
		if err != nil {
			slog.Error("failed to convert event to history item",
				"event_id", env.ID,
				"event_type", env.EventType,
				"error", err,
			)
			continue
		}
		items = append(items, item)
	}

	return &EventHistoryPage{
		Items: items,
		Total: total,
	}, nil
}

func (s *Service) envelopeToHistoryItem(ctx context.Context, env eventstore.EventEnvelope) (EventHistoryItem, error) {
	// Parse payload into map
	var data map[string]any
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return EventHistoryItem{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Remove sensitive fields
	for field := range sensitiveFields {
		delete(data, field)
	}

	// Get actor info
	var actor *ActorInfo
	if actorField, ok := actorFieldMapping[env.EventType]; ok {
		actor = s.resolveActor(ctx, env.EventType, data, actorField)
	}

	return EventHistoryItem{
		ID:         env.ID,
		EventType:  env.EventType,
		Version:    env.Version,
		OccurredAt: env.OccurredAt,
		Actor:      actor,
		Data:       data,
	}, nil
}

func (s *Service) resolveActor(ctx context.Context, eventType string, data map[string]any, actorField string) *ActorInfo {
	actorIDRaw, ok := data[actorField]
	if !ok {
		return nil
	}

	actorIDStr, ok := actorIDRaw.(string)
	if !ok {
		return nil
	}

	actorID, err := uuid.Parse(actorIDStr)
	if err != nil {
		return nil
	}

	// For admin events, return special actor info
	if adminEventTypes[eventType] {
		return &ActorInfo{
			ID:    actorID,
			Name:  "Администратор платформы",
			Email: "",
		}
	}

	// Try to resolve member
	member, err := s.membersProjection.GetByID(ctx, actorID)
	if err != nil {
		// Member might be deleted
		return &ActorInfo{
			ID:    actorID,
			Name:  "Удалённый пользователь",
			Email: "",
		}
	}

	return &ActorInfo{
		ID:    member.ID,
		Name:  member.Name,
		Email: member.Email,
	}
}

// GetDisplayableHistory retrieves paginated event history with human-readable display
func (s *Service) GetDisplayableHistory(ctx context.Context, aggregateID uuid.UUID, aggregateType string, limit, offset int) (*DisplayableHistoryPage, error) {
	envelopes, total, err := s.eventStore.LoadPaginated(ctx, aggregateID, aggregateType, limit, offset)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			return &DisplayableHistoryPage{Items: []DisplayableHistoryItem{}, Total: 0}, nil
		}
		return nil, fmt.Errorf("load events: %w", err)
	}

	// Create resolver for batch operations
	resolver := s.displayRegistry.NewResolver()

	items := make([]DisplayableHistoryItem, 0, len(envelopes))
	for _, env := range envelopes {
		item, err := s.envelopeToDisplayableItem(ctx, env, resolver)
		if err != nil {
			slog.Error("failed to convert event to displayable item",
				slog.String("event_id", env.ID.String()),
				slog.String("event_type", env.EventType),
				slog.String("error", err.Error()),
			)
			continue
		}
		items = append(items, item)
	}

	return &DisplayableHistoryPage{
		Items: items,
		Total: total,
	}, nil
}

func (s *Service) envelopeToDisplayableItem(ctx context.Context, env eventstore.EventEnvelope, resolver *display.CachedResolver) (DisplayableHistoryItem, error) {
	// Parse payload into map for actor resolution
	var data map[string]any
	if err := json.Unmarshal(env.Payload, &data); err != nil {
		return DisplayableHistoryItem{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	// Get actor info
	var actor *ActorInfo
	if actorField, ok := actorFieldMapping[env.EventType]; ok {
		actor = s.resolveActor(ctx, env.EventType, data, actorField)
	}

	// Unmarshal event for display formatting
	evt, err := env.UnmarshalEvent()
	if err != nil {
		return DisplayableHistoryItem{}, fmt.Errorf("unmarshal event: %w", err)
	}

	// Format display
	displayView, err := s.displayRegistry.FormatWithResolver(ctx, evt, resolver)
	if err != nil {
		slog.Warn("failed to format event display, using fallback",
			slog.String("event_type", env.EventType),
			slog.String("error", err.Error()),
		)
		displayView = display.DisplayView{
			Title:       env.EventType,
			Description: "Не удалось отформатировать событие",
			Severity:    "info",
		}
	}

	return DisplayableHistoryItem{
		ID:         env.ID,
		EventType:  env.EventType,
		Version:    env.Version,
		OccurredAt: env.OccurredAt,
		Actor:      actor,
		Display:    displayView,
	}, nil
}
