-- migrations/20261012000000_match_boards_freeze_bg.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- O fundo agora faz parte do retrato do tabuleiro, como a grade: match_boards.bg NULL passa a
-- significar "sem fundo", sempre congelado. Antes, NULL significava "herda o fundo do mapa";
-- as linhas existentes ganham aqui o fundo do mapa de que partiram (se o mapa não tem fundo,
-- continua NULL).
UPDATE match_boards mb
   SET bg = m.bg
  FROM maps m
 WHERE mb.bg IS NULL
   AND m.uuid = mb.map_uuid
   AND m.bg IS NOT NULL;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- No-op: depois do backfill não dá para distinguir uma linha que herdava o fundo de uma que
-- tinha o mesmo fundo gravado de propósito.
SELECT 1;
-- +goose StatementEnd
