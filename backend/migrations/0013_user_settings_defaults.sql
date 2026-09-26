-- +goose Up
-- Perfil de Insistência e Política de Acúmulo padrão do usuário — pré-
-- selecionados ao criar um novo Compromisso (ARCHITECTURE.md §6). Não são
-- lidos pelo motor de cobrança: um Compromisso já grava os seus próprios no
-- momento da criação (commitments.nag_profile/carry_over_policy); estes
-- valores só existem para poupar cliques no formulário.
ALTER TABLE user_settings
    ADD COLUMN default_nag_profile      TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN default_carry_over_policy TEXT NOT NULL DEFAULT 'stack';

-- +goose Down
ALTER TABLE user_settings
    DROP COLUMN default_nag_profile,
    DROP COLUMN default_carry_over_policy;
