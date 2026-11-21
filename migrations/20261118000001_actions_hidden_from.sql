-- migrations/20261118000001_actions_hidden_from.sql
-- +goose Up
-- Quem percebia cada personagem Escondido no instante (status-effects fase 1, PR 2), como lista
-- de permissão: {"<charUUID>": {"effect": "<hiddenEffectUUID>", "perceivedBy": ["<playerUUID>", ...]}}.
-- Um leitor que não está em perceivedBy não o percebia e o histórico esconde dele o personagem
-- (inclusive quem nem era jogador da sessão) até o mestre revelar esse Escondido; o mestre e o
-- dono sempre veem e nunca entram na lista. A linha da action, gravada na abertura do turno,
-- cobre o ator e cada alvo; a de uma reação cobre o reator, na abertura dela (nunca aberta: no
-- fechamento). NULL = nenhum personagem Escondido com lista (nunca '{}').
ALTER TABLE actions ADD COLUMN IF NOT EXISTS hidden_from JSONB;

-- +goose Down
ALTER TABLE actions DROP COLUMN IF EXISTS hidden_from;
