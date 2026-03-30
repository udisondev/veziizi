package support

import (
	"context"
	"errors"
	"fmt"

	"github.com/udisondev/veziizi/backend/internal/domain/support"
	"github.com/udisondev/veziizi/backend/internal/domain/support/entities"
	"github.com/udisondev/veziizi/backend/internal/domain/support/events"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/messaging"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"github.com/udisondev/veziizi/backend/internal/infrastructure/persistence/sequence"
	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
	"github.com/google/uuid"
)

type Service struct {
	db         dbtx.TxManager
	eventStore eventstore.Store
	publisher  *messaging.EventPublisher
	seqGen     *sequence.Generator
}

func NewService(
	db dbtx.TxManager,
	eventStore eventstore.Store,
	publisher *messaging.EventPublisher,
	seqGen *sequence.Generator,
) *Service {
	return &Service{
		db:         db,
		eventStore: eventStore,
		publisher:  publisher,
		seqGen:     seqGen,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*support.Ticket, error) {
	evts, err := s.eventStore.Load(ctx, id, events.AggregateType)
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
			return nil, support.ErrTicketNotFound
		}
		return nil, fmt.Errorf("load ticket: %w", err)
	}
	if len(evts) == 0 {
		return nil, support.ErrTicketNotFound
	}
	return support.NewFromEvents(id, evts), nil
}

type CreateTicketInput struct {
	MemberID       uuid.UUID
	OrgID          uuid.UUID
	Subject        string
	InitialMessage string
}

func (s *Service) CreateTicket(ctx context.Context, input CreateTicketInput) (uuid.UUID, error) {
	var resultID uuid.UUID

	err := s.db.InTx(ctx, func(ctx context.Context) error {
		ticketNumber, err := s.seqGen.NextTicketNumber(ctx)
		if err != nil {
			return fmt.Errorf("get next ticket number: %w", err)
		}

		id := uuid.New()
		t, err := support.New(
			id,
			ticketNumber,
			input.MemberID,
			input.OrgID,
			input.Subject,
			input.InitialMessage,
		)
		if err != nil {
			return err
		}

		if err := s.saveAndPublish(ctx, t); err != nil {
			return err
		}

		resultID = id
		return nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	return resultID, nil
}

type AddUserMessageInput struct {
	TicketID uuid.UUID
	MemberID uuid.UUID
	Content  string
}

func (s *Service) AddUserMessage(ctx context.Context, input AddUserMessageInput) error {
	t, err := s.Get(ctx, input.TicketID)
	if err != nil {
		return err
	}

	if err := t.AddUserMessage(input.MemberID, input.Content); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, t)
}

type AddAdminMessageInput struct {
	TicketID uuid.UUID
	AdminID  uuid.UUID
	Content  string
}

func (s *Service) AddAdminMessage(ctx context.Context, input AddAdminMessageInput) error {
	t, err := s.Get(ctx, input.TicketID)
	if err != nil {
		return err
	}

	if err := t.AddAdminMessage(input.AdminID, input.Content); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, t)
}

type CloseTicketInput struct {
	TicketID   uuid.UUID
	AdminID    uuid.UUID
	Resolution string
}

func (s *Service) CloseTicket(ctx context.Context, input CloseTicketInput) error {
	t, err := s.Get(ctx, input.TicketID)
	if err != nil {
		return err
	}

	if err := t.Close(input.AdminID, input.Resolution); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, t)
}

func (s *Service) ReopenTicket(ctx context.Context, ticketID, memberID uuid.UUID) error {
	t, err := s.Get(ctx, ticketID)
	if err != nil {
		return err
	}

	if err := t.Reopen(memberID); err != nil {
		return err
	}

	return s.saveAndPublish(ctx, t)
}

func (s *Service) saveAndPublish(ctx context.Context, t *support.Ticket) error {
	changes := t.Changes()
	if len(changes) == 0 {
		return nil
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		if err := s.eventStore.Save(ctx, changes...); err != nil {
			return fmt.Errorf("save events: %w", err)
		}

		if err := s.publisher.Publish(ctx, "support.events", changes...); err != nil {
			return fmt.Errorf("publish events: %w", err)
		}

		t.ClearChanges()
		return nil
	})
}

// MessageResponse is a DTO for returning messages to API
type MessageResponse struct {
	ID         uuid.UUID           `json:"id"`
	SenderType entities.SenderType `json:"sender_type"`
	SenderID   uuid.UUID           `json:"sender_id"`
	Content    string              `json:"content"`
	CreatedAt  string              `json:"created_at"`
}

// TicketResponse is a DTO for returning ticket details to API
type TicketResponse struct {
	ID           uuid.UUID         `json:"id"`
	TicketNumber int64             `json:"ticket_number"`
	Subject      string            `json:"subject"`
	Status       string            `json:"status"`
	Messages     []MessageResponse `json:"messages"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	ClosedAt     *string           `json:"closed_at,omitempty"`
}
