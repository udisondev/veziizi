package sequence

import (
	"context"
	"fmt"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
)

// Generator provides atomic sequence number generation using PostgreSQL sequences
type Generator struct {
	db dbtx.TxManager
}

// NewGenerator creates a new sequence generator
func NewGenerator(db dbtx.TxManager) *Generator {
	return &Generator{db: db}
}

// NextOrderNumber returns the next order number from the sequence
func (g *Generator) NextOrderNumber(ctx context.Context) (int64, error) {
	var num int64
	err := g.db.QueryRow(ctx, "SELECT nextval('order_number_seq')").Scan(&num)
	if err != nil {
		return 0, fmt.Errorf("get next order number: %w", err)
	}
	return num, nil
}

// NextRequestNumber returns the next freight request number from the sequence
func (g *Generator) NextRequestNumber(ctx context.Context) (int64, error) {
	var num int64
	err := g.db.QueryRow(ctx, "SELECT nextval('request_number_seq')").Scan(&num)
	if err != nil {
		return 0, fmt.Errorf("get next request number: %w", err)
	}
	return num, nil
}

// NextTicketNumber returns the next support ticket number from the sequence
func (g *Generator) NextTicketNumber(ctx context.Context) (int64, error) {
	var num int64
	err := g.db.QueryRow(ctx, "SELECT nextval('ticket_number_seq')").Scan(&num)
	if err != nil {
		return 0, fmt.Errorf("get next ticket number: %w", err)
	}
	return num, nil
}
