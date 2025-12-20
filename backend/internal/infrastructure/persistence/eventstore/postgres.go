package eventstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"codeberg.org/udison/veziizi/backend/internal/pkg/dbtx"
	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultSnapshotThreshold = 100
	uniqueViolationCode      = "23505"
)

type PostgresStore struct {
	db                dbtx.TxManager
	snapshotThreshold int64
	psql              squirrel.StatementBuilderType
}

type PostgresStoreOption func(*PostgresStore)

func WithSnapshotThreshold(threshold int64) PostgresStoreOption {
	return func(s *PostgresStore) {
		s.snapshotThreshold = threshold
	}
}

func NewPostgresStore(db dbtx.TxManager, opts ...PostgresStoreOption) *PostgresStore {
	s := &PostgresStore{
		db:                db,
		snapshotThreshold: defaultSnapshotThreshold,
		psql:              squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type aggregateKey struct {
	ID   uuid.UUID
	Type string
}

func (s *PostgresStore) Save(ctx context.Context, events ...Event) error {
	if len(events) == 0 {
		return nil
	}

	return s.db.InTx(ctx, func(ctx context.Context) error {
		grouped := make(map[aggregateKey][]Event)
		for _, event := range events {
			key := aggregateKey{ID: event.AggregateID(), Type: event.AggregateType()}
			grouped[key] = append(grouped[key], event)
		}

		for key, aggEvents := range grouped {
			var lastVersion int64

			for _, event := range aggEvents {
				envelope, err := NewEventEnvelope(event, nil)
				if err != nil {
					return fmt.Errorf("failed to create envelope: %w", err)
				}

				metadataJSON, err := json.Marshal(envelope.Metadata)
				if err != nil {
					return fmt.Errorf("failed to marshal metadata: %w", err)
				}

				query, args, err := s.psql.
					Insert("events").
					Columns("id", "aggregate_id", "aggregate_type", "event_type", "version", "data", "metadata", "occurred_at").
					Values(
						envelope.ID,
						envelope.AggregateID,
						envelope.AggregateType,
						envelope.EventType,
						envelope.Version,
						envelope.Payload,
						metadataJSON,
						envelope.OccurredAt,
					).
					ToSql()
				if err != nil {
					return fmt.Errorf("failed to build insert query: %w", err)
				}

				if _, err := s.db.Exec(ctx, query, args...); err != nil {
					var pgErr *pgconn.PgError
					if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
						return ErrConcurrentModification
					}
					return fmt.Errorf("failed to insert event: %w", err)
				}

				lastVersion = event.Version()
			}

			if err := s.maybeCreateSnapshot(ctx, key.ID, key.Type, lastVersion); err != nil {
				return fmt.Errorf("failed to create snapshot: %w", err)
			}
		}

		return nil
	})
}

func (s *PostgresStore) Load(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]Event, error) {
	snapshot, err := s.loadSnapshot(ctx, aggregateID, aggregateType)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot: %w", err)
	}

	var fromVersion int64 = 0
	if snapshot != nil {
		fromVersion = snapshot.Version
	}

	query, args, err := s.psql.
		Select("id", "aggregate_id", "aggregate_type", "event_type", "version", "data", "metadata", "occurred_at").
		From("events").
		Where(squirrel.And{
			squirrel.Eq{"aggregate_id": aggregateID},
			squirrel.Eq{"aggregate_type": aggregateType},
			squirrel.Gt{"version": fromVersion},
		}).
		OrderBy("version ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var dbRows []eventRow
	if err := pgxscan.ScanAll(&dbRows, rows); err != nil {
		return nil, fmt.Errorf("failed to scan events: %w", err)
	}

	if len(dbRows) == 0 && snapshot == nil {
		return nil, ErrAggregateNotFound
	}

	events := make([]Event, 0, len(dbRows))
	for _, row := range dbRows {
		envelope := row.toEnvelope()
		event, err := envelope.UnmarshalEvent()
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal event %s: %w", row.EventType, err)
		}
		events = append(events, event)
	}

	return events, nil
}

func (s *PostgresStore) loadSnapshot(ctx context.Context, aggregateID uuid.UUID, aggregateType string) (*Snapshot, error) {
	query, args, err := s.psql.
		Select("aggregate_id", "aggregate_type", "version", "data").
		From("snapshots").
		Where(squirrel.Eq{
			"aggregate_id":   aggregateID,
			"aggregate_type": aggregateType,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build snapshot query: %w", err)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshot: %w", err)
	}
	defer rows.Close()

	var snapshots []Snapshot
	if err := pgxscan.ScanAll(&snapshots, rows); err != nil {
		return nil, fmt.Errorf("failed to scan snapshot: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	return &snapshots[0], nil
}

func (s *PostgresStore) maybeCreateSnapshot(ctx context.Context, aggregateID uuid.UUID, aggregateType string, currentVersion int64) error {
	if currentVersion%s.snapshotThreshold != 0 {
		return nil
	}

	envelopes, err := s.loadAllEnvelopes(ctx, aggregateID, aggregateType)
	if err != nil {
		return fmt.Errorf("failed to load events for snapshot: %w", err)
	}

	snapshotData, err := json.Marshal(envelopes)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot data: %w", err)
	}

	query, args, err := s.psql.
		Insert("snapshots").
		Columns("aggregate_id", "aggregate_type", "version", "data", "created_at").
		Values(aggregateID, aggregateType, currentVersion, snapshotData, squirrel.Expr("NOW()")).
		Suffix("ON CONFLICT (aggregate_id) DO UPDATE SET version = EXCLUDED.version, data = EXCLUDED.data, created_at = NOW()").
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build snapshot upsert query: %w", err)
	}

	if _, err := s.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to upsert snapshot: %w", err)
	}

	return nil
}

func (s *PostgresStore) loadAllEnvelopes(ctx context.Context, aggregateID uuid.UUID, aggregateType string) ([]EventEnvelope, error) {
	query, args, err := s.psql.
		Select("id", "aggregate_id", "aggregate_type", "event_type", "version", "data", "metadata", "occurred_at").
		From("events").
		Where(squirrel.Eq{
			"aggregate_id":   aggregateID,
			"aggregate_type": aggregateType,
		}).
		OrderBy("version ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var dbRows []eventRow
	if err := pgxscan.ScanAll(&dbRows, rows); err != nil {
		return nil, fmt.Errorf("failed to scan events: %w", err)
	}

	envelopes := make([]EventEnvelope, 0, len(dbRows))
	for _, row := range dbRows {
		envelopes = append(envelopes, row.toEnvelope())
	}

	return envelopes, nil
}

type eventRow struct {
	ID            uuid.UUID `db:"id"`
	AggregateID   uuid.UUID `db:"aggregate_id"`
	AggregateType string    `db:"aggregate_type"`
	EventType     string    `db:"event_type"`
	Version       int64     `db:"version"`
	Data          []byte    `db:"data"`
	Metadata      []byte    `db:"metadata"`
	OccurredAt    time.Time `db:"occurred_at"`
}

func (r eventRow) toEnvelope() EventEnvelope {
	var metadata map[string]string
	if len(r.Metadata) > 0 {
		// SEC-018: Логируем ошибки unmarshal вместо игнорирования
		if err := json.Unmarshal(r.Metadata, &metadata); err != nil {
			slog.Error("SEC-018: failed to unmarshal event metadata",
				slog.String("event_id", r.ID.String()),
				slog.String("error", err.Error()))
		}
	}

	return EventEnvelope{
		ID:            r.ID,
		AggregateID:   r.AggregateID,
		AggregateType: r.AggregateType,
		EventType:     r.EventType,
		Version:       r.Version,
		Payload:       r.Data,
		Metadata:      metadata,
		OccurredAt:    r.OccurredAt,
	}
}

type Snapshot struct {
	AggregateID   uuid.UUID `db:"aggregate_id"`
	AggregateType string    `db:"aggregate_type"`
	Version       int64     `db:"version"`
	Data          []byte    `db:"data"`
}

func (s *PostgresStore) LoadPaginated(ctx context.Context, aggregateID uuid.UUID, aggregateType string, limit, offset int) ([]EventEnvelope, int, error) {
	// Get total count
	countQuery, countArgs, err := s.psql.
		Select("COUNT(*)").
		From("events").
		Where(squirrel.Eq{
			"aggregate_id":   aggregateID,
			"aggregate_type": aggregateType,
		}).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build count query: %w", err)
	}

	var total int
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	if total == 0 {
		return nil, 0, ErrAggregateNotFound
	}

	// Get paginated events (newest first)
	query, args, err := s.psql.
		Select("id", "aggregate_id", "aggregate_type", "event_type", "version", "data", "metadata", "occurred_at").
		From("events").
		Where(squirrel.Eq{
			"aggregate_id":   aggregateID,
			"aggregate_type": aggregateType,
		}).
		OrderBy("version DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build select query: %w", err)
	}

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var dbRows []eventRow
	if err := pgxscan.ScanAll(&dbRows, rows); err != nil {
		return nil, 0, fmt.Errorf("failed to scan events: %w", err)
	}

	envelopes := make([]EventEnvelope, 0, len(dbRows))
	for _, row := range dbRows {
		envelopes = append(envelopes, row.toEnvelope())
	}

	return envelopes, total, nil
}
