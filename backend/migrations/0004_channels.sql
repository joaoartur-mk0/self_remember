-- +goose Up
-- config fica presente desde já (forma decidida em ARCHITECTURE.md §4), mas
-- sem cifra implementada nesta fase: só o canal falso de log é usado até a
-- fase de Canais reais, que é quem grava e lê esse campo de verdade.
CREATE TABLE channels (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind           TEXT NOT NULL,
    address        TEXT NOT NULL,
    config         BYTEA,
    verified_at    TIMESTAMPTZ,
    fallback_order INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_channels_user_id_fallback_order ON channels (user_id, fallback_order);

-- +goose Down
DROP TABLE channels;
