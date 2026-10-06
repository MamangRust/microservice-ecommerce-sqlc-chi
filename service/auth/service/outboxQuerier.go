package service

import (
	"context"
	"time"

	db "github.com/MamangRust/microservice-ecommerce-auth/database/schema"
	"github.com/MamangRust/microservice-ecommerce-pkg/outbox"
	"github.com/jackc/pgx/v5/pgtype"
)

// outboxQuerier adapts the auth service's generated db.Queries to the
// schema-agnostic outbox.OutboxQuerier contract (per-service schema). The
// adapter is bound either to the service pool (relay + non-atomic enqueue) or
// to a transaction (EnqueueInTx).
//
// Note: the auth migrations do not own an outbox_events table (the queries are
// type-checked via database/schema-overlay), so at runtime these methods only
// succeed against a database that actually has the table (e.g. the shared test
// database); production wiring keeps the outbox service nil and falls back to
// direct Kafka.
type outboxQuerier struct {
	db *db.Queries
}

// NewOutboxQuerier builds the outbox adapter. Pass queries == nil when the
// service database has no outbox_events table (enqueue then no-ops).
func NewOutboxQuerier(queries *db.Queries) outbox.OutboxQuerier {
	if queries == nil {
		return nil
	}
	return &outboxQuerier{db: queries}
}

func toOutboxEvent(row *db.OutboxEvent) outbox.OutboxEvent {
	return outbox.OutboxEvent{
		OutboxID:      row.OutboxID,
		Topic:         row.Topic,
		EventKey:      row.EventKey,
		Payload:       row.Payload,
		Status:        row.Status,
		Attempts:      row.Attempts,
		NextAttemptAt: row.NextAttemptAt,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}

func (q *outboxQuerier) CreateOutboxEvent(ctx context.Context, params outbox.CreateOutboxEventParams) (outbox.OutboxEvent, error) {
	row, err := q.db.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		Topic:    params.Topic,
		EventKey: params.EventKey,
		Payload:  params.Payload,
	})
	if err != nil {
		return outbox.OutboxEvent{}, err
	}
	return toOutboxEvent(row), nil
}

func (q *outboxQuerier) ClaimPendingOutboxEvents(ctx context.Context, params outbox.ClaimPendingOutboxEventsParams) ([]outbox.OutboxEvent, error) {
	rows, err := q.db.ClaimPendingOutboxEvents(ctx, db.ClaimPendingOutboxEventsParams{
		Limit:         params.Limit,
		NextAttemptAt: params.NextAttemptAt,
	})
	if err != nil {
		return nil, err
	}
	events := make([]outbox.OutboxEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, toOutboxEvent(row))
	}
	return events, nil
}

func (q *outboxQuerier) MarkOutboxEventFailed(ctx context.Context, params outbox.MarkOutboxEventFailedParams) (outbox.OutboxEvent, error) {
	row, err := q.db.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
		OutboxID:      params.OutboxID,
		NextAttemptAt: params.NextAttemptAt,
	})
	if err != nil {
		return outbox.OutboxEvent{}, err
	}
	return toOutboxEvent(row), nil
}

func (q *outboxQuerier) MarkOutboxEventDelivered(ctx context.Context, outboxID int64) (outbox.OutboxEvent, error) {
	row, err := q.db.MarkOutboxEventDelivered(ctx, outboxID)
	if err != nil {
		return outbox.OutboxEvent{}, err
	}
	return toOutboxEvent(row), nil
}

func (q *outboxQuerier) MarkOutboxEventDead(ctx context.Context, outboxID int64) (outbox.OutboxEvent, error) {
	row, err := q.db.MarkOutboxEventDead(ctx, outboxID)
	if err != nil {
		return outbox.OutboxEvent{}, err
	}
	return toOutboxEvent(row), nil
}

func (q *outboxQuerier) DeleteOldOutboxEvents(ctx context.Context, cutoff time.Time) (int64, error) {
	return q.db.DeleteOldOutboxEvents(ctx, pgtype.Timestamp{Time: cutoff, Valid: true})
}
