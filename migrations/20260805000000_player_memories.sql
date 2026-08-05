-- migrations/20260805000000_player_memories.sql
-- Renames the fog memory table to match what it now stores. The old table held a set
-- of explored grid cells; it now holds the set of static map features the player has
-- observed. No data migration: player_fog_states was never written to (the repository
-- was a stub), so the rename is safe as-is.
-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE player_fog_states RENAME TO player_memories;
ALTER TABLE player_memories RENAME COLUMN explored TO seen_features;
ALTER TABLE player_memories DROP COLUMN grid_kind;
ALTER INDEX idx_player_fog_states_match_map RENAME TO idx_player_memories_match_map;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER INDEX idx_player_memories_match_map RENAME TO idx_player_fog_states_match_map;
ALTER TABLE player_memories ADD COLUMN grid_kind VARCHAR(10) NOT NULL DEFAULT 'square';
ALTER TABLE player_memories RENAME COLUMN seen_features TO explored;
ALTER TABLE player_memories RENAME TO player_fog_states;

COMMIT;
-- +goose StatementEnd
