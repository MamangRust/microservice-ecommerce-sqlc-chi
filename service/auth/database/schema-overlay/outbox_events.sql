-- sqlc-only schema overlay (NOT applied by goose).
--
-- The auth service references the outbox contract (EnqueueInTx in the
-- forgot-password flow) but its migrations do not own the outbox_events table.
-- This overlay exists only so sqlc can type-check those queries; it is never
-- mounted as a migration.
CREATE TABLE "outbox_events" (
    "outbox_id" BIGSERIAL PRIMARY KEY,
    "topic" VARCHAR(255) NOT NULL,
    "event_key" VARCHAR(255) NOT NULL,
    "payload" JSONB NOT NULL,
    "status" VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK ("status" IN ('pending', 'delivered', 'dead')),
    "attempts" INT NOT NULL DEFAULT 0,
    "next_attempt_at" TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
