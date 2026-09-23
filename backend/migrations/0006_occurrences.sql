-- +goose Up
-- Ver ADR-0002: Ocorrências são materializadas, não calculadas sob demanda.
-- UNIQUE (commitment_id, due_at) é o que torna o Materializador idempotente.
CREATE TABLE occurrences (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    commitment_id UUID NOT NULL REFERENCES commitments(id) ON DELETE CASCADE,
    due_at        TIMESTAMPTZ NOT NULL,
    -- IANA que gerou due_at (ARCHITECTURE.md §4) — para auditoria do cálculo,
    -- não para recalculá-lo: due_at já é o instante final em UTC.
    timezone      TEXT NOT NULL,
    state         TEXT NOT NULL DEFAULT 'pending',
    current_step  INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (commitment_id, due_at)
);

-- Consulta do Scanner: ocorrências não terminais vencidas.
CREATE INDEX idx_occurrences_due_at ON occurrences (due_at) WHERE state NOT IN ('done', 'dismissed', 'abandoned');

-- +goose Down
DROP TABLE occurrences;
