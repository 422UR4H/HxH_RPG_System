-- migrations/20260616000000_fog_state.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE maps ADD COLUMN fog_mode VARCHAR(10) NOT NULL DEFAULT 'explored';

CREATE TABLE IF NOT EXISTS player_fog_states (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id   UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
    map_id     UUID        NOT NULL REFERENCES maps(uuid)    ON DELETE CASCADE,
    player_id  UUID        NOT NULL,
    grid_kind  VARCHAR(10) NOT NULL,
    explored   JSONB       NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (match_id, map_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_player_fog_states_match_map ON player_fog_states(match_id, map_id);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_player_fog_states_match_map;
DROP TABLE IF EXISTS player_fog_states;
ALTER TABLE maps DROP COLUMN IF EXISTS fog_mode;

COMMIT;
-- +goose StatementEnd
