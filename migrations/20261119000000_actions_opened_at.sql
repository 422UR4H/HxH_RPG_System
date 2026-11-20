-- migrations/20261119000000_actions_opened_at.sql
-- +goose Up
-- O instante real em que a linha abriu (status-effects fase 1, PR 2, Task 15b): a da action é
-- a abertura do turno (turns.created_at); a de uma reação, quando o mestre a abriu. NULL = nunca
-- aberta (uma reação sentada ou anexada que fechou sem a palavra) ou linha de antes desta coluna.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS opened_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE actions DROP COLUMN IF EXISTS opened_at;
