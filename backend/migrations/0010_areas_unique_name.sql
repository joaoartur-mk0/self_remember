-- +goose Up
-- Duas Áreas com o mesmo nome para o mesmo usuário não fazem sentido — o
-- usuário não teria como distingui-las num seletor. Mesma regra já aplicada
-- a labels (0008_labels.sql); a migration original de areas (0003) ficou
-- deliberadamente mínima e adia isso para a fase de CRUD, que é agora.
ALTER TABLE areas ADD CONSTRAINT areas_user_id_name_key UNIQUE (user_id, name);

-- +goose Down
ALTER TABLE areas DROP CONSTRAINT areas_user_id_name_key;
