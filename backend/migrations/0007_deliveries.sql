-- +goose Up
-- UNIQUE (occurrence_id, step_index, channel_id) é a idempotência do
-- Entregador (ARCHITECTURE.md §3.3): a linha é inserida antes da chamada
-- externa, e uma colisão aqui significa "já entregue", não erro.
CREATE TABLE deliveries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurrence_id       UUID NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    step_index          INT NOT NULL,
    channel_id          UUID NOT NULL REFERENCES channels(id),
    correlation_id      TEXT NOT NULL,
    provider_message_id TEXT,
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (occurrence_id, step_index, channel_id)
);

CREATE INDEX idx_deliveries_occurrence_id ON deliveries (occurrence_id);

-- +goose Down
DROP TABLE deliveries;
