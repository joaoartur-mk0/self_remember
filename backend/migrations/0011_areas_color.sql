-- +goose Up
-- Cor da Área, prometida pela migration 0003 para "a fase de CRUD (API)".
-- Opcional: uma Área sem cor ainda é uma Área válida, o frontend decide o
-- que fazer com a ausência (cor padrão por posição, por exemplo).
ALTER TABLE areas ADD COLUMN color TEXT;

-- +goose Down
ALTER TABLE areas DROP COLUMN color;
