-- +goose Up
-- Um Canal só pode receber Entregas depois de Verificado (ARCHITECTURE.md
-- §7). O código é hasheado como um segredo de vida curta, mesmo sendo
-- numérico e de poucos dígitos.
ALTER TABLE channels
    ADD COLUMN verification_code_hash   TEXT,
    ADD COLUMN verification_expires_at  TIMESTAMPTZ;

-- +goose Down
ALTER TABLE channels
    DROP COLUMN verification_code_hash,
    DROP COLUMN verification_expires_at;
