-- +goose Up
-- Ver ADR-0008: rotação com revogação de toda a sessão do usuário no reuso.
-- token_hash guarda sha256(token) — o valor bruto nunca é persistido, igual a senha.
-- rotated_to_id é só trilha de auditoria; a revogação por reuso opera em user_id, não
-- em cadeia, então não depende dessa coluna para funcionar.
CREATE TABLE refresh_tokens (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    TEXT NOT NULL UNIQUE,
    rotated_at    TIMESTAMPTZ,
    rotated_to_id UUID REFERENCES refresh_tokens(id),
    revoked_at    TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- +goose Down
DROP TABLE refresh_tokens;
