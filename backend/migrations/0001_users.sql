-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- user_settings.timezone é IANA (ex.: "America/Sao_Paulo") e obrigatório: a expansão
-- de recorrência e a Janela de Silêncio dependem do fuso do usuário, nunca de UTC.
CREATE TABLE user_settings (
    user_id           UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    timezone          TEXT NOT NULL,
    quiet_hours_start TIME,
    quiet_hours_end   TIME,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_settings;
DROP TABLE users;
