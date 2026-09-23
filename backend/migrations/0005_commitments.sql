-- +goose Up
CREATE TABLE commitments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    area_id           UUID NOT NULL REFERENCES areas(id),
    title             TEXT NOT NULL,
    -- DTSTART (RFC 5545): data/hora de parede do primeiro vencimento. Ver
    -- internal/domain/commitment/recurrence.go.
    recurrence_start  TIMESTAMPTZ NOT NULL,
    -- RRULE (RFC 5545); vazio significa Compromisso único.
    recurrence_rule   TEXT NOT NULL DEFAULT '',
    nag_profile       TEXT NOT NULL,
    carry_over_policy TEXT NOT NULL,
    archived_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_commitments_user_id ON commitments (user_id);
-- O Materializador varre Compromissos ativos a cada hora; este índice existe
-- para essa varredura, não para consulta por usuário.
CREATE INDEX idx_commitments_active ON commitments (id) WHERE archived_at IS NULL;

-- +goose Down
DROP TABLE commitments;
