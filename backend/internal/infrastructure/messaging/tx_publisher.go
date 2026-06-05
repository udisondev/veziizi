package messaging

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/udisondev/veziizi/backend/internal/pkg/dbtx"
)

// txAwarePublisher реализует message.Publisher и сам выбирает sql-publisher
// исходя из ctx первого сообщения: tx из dbtx.FromCtx, иначе default на pool.
//
// Это позволяет завернуть его в cqrs.EventBus без потери tx-aware outbox-семантики:
// EventBus сначала Marshal'ит, ставит ctx в msg, потом зовёт publisher.Publish(topic, msg).
// Мы здесь читаем msg.Context() и решаем, какой sql-publisher использовать.
type txAwarePublisher struct {
	pool      *pgxpool.Pool
	defaultPub message.Publisher
	logger    watermill.LoggerAdapter
}

func newTxAwarePublisher(pool *pgxpool.Pool, logger watermill.LoggerAdapter) (*txAwarePublisher, error) {
	defaultPub, err := sql.NewPublisher(
		sql.BeginnerFromPgx(pool),
		sql.PublisherConfig{
			SchemaAdapter:        sql.DefaultPostgreSQLSchema{},
			AutoInitializeSchema: true,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create default publisher: %w", err)
	}
	return &txAwarePublisher{pool: pool, defaultPub: defaultPub, logger: logger}, nil
}

// Publish — все сообщения в одном вызове предполагаются опубликованными в одном
// ctx (так и работает cqrs.EventBus: один msg на один Publish). Если в ctx есть
// активная транзакция — публикуем через sql.TxFromPgx(tx), иначе через default
// publisher на пуле в autocommit. AutoInitializeSchema выключен на tx-пути,
// потому что CREATE TABLE вызывает implicit commit и ломает транзакцию.
func (p *txAwarePublisher) Publish(topic string, messages ...*message.Message) error {
	if len(messages) == 0 {
		return nil
	}

	ctx := messages[0].Context()
	tx, hasTx := dbtx.FromCtx(ctx)
	if !hasTx {
		return p.defaultPub.Publish(topic, messages...)
	}

	txPub, err := sql.NewPublisher(
		sql.TxFromPgx(tx),
		sql.PublisherConfig{
			SchemaAdapter: sql.DefaultPostgreSQLSchema{},
		},
		p.logger,
	)
	if err != nil {
		return fmt.Errorf("create tx publisher: %w", err)
	}
	defer func() {
		if cerr := txPub.Close(); cerr != nil {
			p.logger.Error("failed to close tx publisher", cerr, nil)
		}
	}()

	if err := txPub.Publish(topic, messages...); err != nil {
		return fmt.Errorf("publish in tx: %w", err)
	}
	return nil
}

// Close закрывает только default publisher; tx-publisher'ы создаются по
// требованию и закрываются на месте.
func (p *txAwarePublisher) Close() error {
	return p.defaultPub.Close()
}

// DefaultPublisher возвращает sql-publisher на пуле в autocommit-режиме.
// Используется PoisonQueue middleware (DLQ всегда вне tx) и любыми
// non-event-store топиками вроде notification.email.
func (p *txAwarePublisher) DefaultPublisher() message.Publisher {
	return p.defaultPub
}
