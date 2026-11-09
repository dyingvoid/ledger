BEGIN;

CREATE TABLE ledger.idempotency (
    key TEXT PRIMARY KEY CHECK (length(key) BETWEEN 1 AND 255),
    status TEXT NOT NULL CHECK (status IN ('in_progress', 'success', 'fail')),
    result JSONB,
    error TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idempotency_expires_at_idx ON ledger.idempotency (expires_at);

COMMIT;
