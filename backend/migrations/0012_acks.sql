-- +goose Up
-- Ack é o registro de auditoria de um Reconhecimento (CONTEXT.md): quando e
-- como o usuário reagiu a uma Ocorrência. Nunca há UPDATE/DELETE nesta
-- tabela — é log, só INSERT (ARCHITECTURE.md §4, ADR-0006).
CREATE TABLE acks (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurrence_id  UUID NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    kind           TEXT NOT NULL,
    source         TEXT NOT NULL DEFAULT 'web',
    channel_id     UUID REFERENCES channels(id),
    correlation_id TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_acks_occurrence_id ON acks (occurrence_id);

-- +goose Down
DROP TABLE acks;
