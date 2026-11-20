-- migrations/20261120000001_cancelled_actions_hidden_from.sql
-- +goose Up
-- O registro do ator Escondido no instante do cancelamento (revisão final do PR 2): a mesma
-- forma de actions.hidden_from, só com o ator. NULL = o ator não estava Escondido com lista que
-- não é "todos" (nunca '{}'). O histórico tira a action cancelada de quem não o percebia até o
-- mestre revelar o Escondido.
ALTER TABLE cancelled_actions ADD COLUMN IF NOT EXISTS hidden_from JSONB;

-- A forma das duas colunas é uma lista de permissão e substitui a descrita no comentário da
-- 20261118000001 ({charUUID: {playerUUID: efeito}}), que não vale mais (sem dado gravado
-- nela: pré-produção, a coluna nasceu neste mesmo PR).
COMMENT ON COLUMN actions.hidden_from IS
  '{"<charUUID>": {"effect": "<hiddenEffectUUID>", "perceivedBy": ["<playerUUID>", ...]}} por personagem Escondido com lista que não é "todos": o efeito e quem o percebia (lista de permissão). NULL = nenhum. O mestre e o dono são decididos na leitura.';
COMMENT ON COLUMN cancelled_actions.hidden_from IS
  'A forma de actions.hidden_from, só com o ator, no instante do cancelamento. NULL = não estava Escondido.';

-- +goose Down
COMMENT ON COLUMN actions.hidden_from IS NULL;
ALTER TABLE cancelled_actions DROP COLUMN IF EXISTS hidden_from;
