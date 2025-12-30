package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/infrastructure/persistence/eventstore"
	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventPublisher struct {
	pool             *pgxpool.Pool
	defaultPublisher message.Publisher
	logger           watermill.LoggerAdapter
	config           sql.PublisherConfig
}

func NewEventPublisher(pool *pgxpool.Pool, logger watermill.LoggerAdapter) (*EventPublisher, error) {
	config := sql.PublisherConfig{
		SchemaAdapter:        sql.DefaultPostgreSQLSchema{},
		AutoInitializeSchema: true,
	}

	defaultPublisher, err := sql.NewPublisher(
		sql.BeginnerFromPgx(pool),
		config,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create default publisher: %w", err)
	}

	return &EventPublisher{
		pool:             pool,
		defaultPublisher: defaultPublisher,
		logger:           logger,
		config:           config,
	}, nil
}

func (p *EventPublisher) Publish(ctx context.Context, topic string, events ...eventstore.Event) error {
	messages := make([]*message.Message, 0, len(events))

	for _, event := range events {
		envelope, err := eventstore.NewEventEnvelope(event, nil)
		if err != nil {
			return fmt.Errorf("failed to create envelope: %w", err)
		}

		payload, err := json.Marshal(envelope)
		if err != nil {
			return fmt.Errorf("failed to marshal envelope: %w", err)
		}

		msg := message.NewMessage(uuid.New().String(), payload)
		msg.Metadata.Set("aggregate_id", event.AggregateID().String())
		msg.Metadata.Set("aggregate_type", event.AggregateType())
		msg.Metadata.Set("event_type", event.EventType())

		messages = append(messages, msg)
	}

	// Check if we have a transaction in context
	if tx, ok := dbtx.FromCtx(ctx); ok {
		// Create publisher for this transaction
		txPublisher, err := sql.NewPublisher(
			sql.TxFromPgx(tx),
			sql.PublisherConfig{
				SchemaAdapter: sql.DefaultPostgreSQLSchema{},
				// No AutoInitializeSchema for tx - it causes implicit commit
			},
			p.logger,
		)
		if err != nil {
			return fmt.Errorf("failed to create tx publisher: %w", err)
		}
		// CRITICAL: Close txPublisher to prevent resource leak
		defer func() {
			if err := txPublisher.Close(); err != nil {
				// Log but don't fail - transaction already handled
				p.logger.Error("failed to close tx publisher", err, nil)
			}
		}()

		if err := txPublisher.Publish(topic, messages...); err != nil {
			return fmt.Errorf("failed to publish messages in tx: %w", err)
		}

		return nil
	}

	// Use default publisher without transaction
	if err := p.defaultPublisher.Publish(topic, messages...); err != nil {
		return fmt.Errorf("failed to publish messages: %w", err)
	}

	return nil
}

func (p *EventPublisher) Close() error {
	return p.defaultPublisher.Close()
}

// RawPublisher возвращает underlying watermill publisher для отправки raw messages
func (p *EventPublisher) RawPublisher() message.Publisher {
	return p.defaultPublisher
}
