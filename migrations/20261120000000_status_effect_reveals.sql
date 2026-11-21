-- migrations/20261120000000_status_effect_reveals.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- As revelações do mestre (Task 16): cada linha torna visível a todos os jogadores, no histórico,
-- o que o efeito escondeu até revealed_at. Cada comando insere uma linha nova (uma revelação
-- posterior cobre os eventos posteriores); o histórico lê o MAX(revealed_at) por efeito, em
-- qualquer partida.
--
-- effect_uuid sem FK: o efeito pode já ter sido removido da ficha; quem o identifica é o
-- status_effect_events, que o gateway consulta antes de revelar.
CREATE TABLE IF NOT EXISTS status_effect_reveals (
  uuid        UUID        PRIMARY KEY,
  effect_uuid UUID        NOT NULL,
  match_uuid  UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  revealed_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_status_effect_reveals_effect ON status_effect_reveals(effect_uuid);
CREATE INDEX IF NOT EXISTS idx_status_effect_reveals_match  ON status_effect_reveals(match_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;
DROP TABLE IF EXISTS status_effect_reveals;
COMMIT;
-- +goose StatementEnd
