-- +goose Up
-- Descrição explica o que o Compromisso é — o Título sozinho reconhece
-- ("Remédio"), a Descrição diz o quê ("Losartana, 1 comprimido, com água").
-- Aparece abaixo do título e da Área na mensagem de Cobrança
-- (internal/jobs/entregador.go). NOT NULL DEFAULT '' para não quebrar
-- Compromissos já existentes; o domínio exige um valor não vazio em
-- Compromissos novos (commitment.ErrDescriptionRequired).
ALTER TABLE commitments ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE commitments DROP COLUMN description;
