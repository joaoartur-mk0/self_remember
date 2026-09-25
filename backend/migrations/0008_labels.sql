-- +goose Up
-- Etiqueta: classificação livre e transversal, sem efeito na cobrança
-- (ARCHITECTURE.md §4, CONTEXT.md).
CREATE TABLE labels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);

CREATE TABLE commitment_labels (
    commitment_id UUID NOT NULL REFERENCES commitments(id) ON DELETE CASCADE,
    label_id      UUID NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (commitment_id, label_id)
);

-- +goose Down
DROP TABLE commitment_labels;
DROP TABLE labels;
