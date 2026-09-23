-- +goose Up
-- Forma mínima por ora: só o necessário para commitments.area_id ter para onde
-- apontar. Janela de Silêncio por Área e cor ficam para a fase de CRUD (API).
CREATE TABLE areas (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_areas_user_id ON areas (user_id);

-- +goose Down
DROP TABLE areas;
