-- migrations/20261118000001_actions_hidden_from.sql
-- +goose Up
-- Para quem cada personagem estava Escondido no instante (status-effects fase 1, PR 2):
-- {"<charUUID>": {"<playerUUID>": "<hiddenEffectUUID>"}}. A linha da action, gravada na abertura
-- do turno, cobre o ator e cada alvo; a de uma reação cobre o reator, na abertura dela (nunca
-- aberta: no fechamento). NULL = ninguém (nunca '{}': quase nada fica escondido). O mestre e o
-- dono nunca entram. O histórico esconde do leitor o que o mapa nomeia para ele até o mestre
-- revelar o Escondido.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS hidden_from JSONB;

-- +goose Down
ALTER TABLE actions DROP COLUMN IF EXISTS hidden_from;
