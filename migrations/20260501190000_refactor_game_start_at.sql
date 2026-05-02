-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE matches ADD COLUMN game_scheduled_at TIMESTAMP;
UPDATE matches SET game_scheduled_at = game_start_at;
ALTER TABLE matches ALTER COLUMN game_scheduled_at SET NOT NULL;

ALTER TABLE matches ALTER COLUMN game_start_at DROP NOT NULL;
UPDATE matches SET game_start_at = NULL;

DROP INDEX IF EXISTS idx_matches_is_public_game_start_master;
DROP INDEX IF EXISTS idx_matches_game_start_at;

CREATE INDEX IF NOT EXISTS idx_matches_is_public_game_scheduled_master ON matches(is_public, game_scheduled_at, master_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_game_scheduled_at ON matches(game_scheduled_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

UPDATE matches SET game_start_at = game_scheduled_at WHERE game_start_at IS NULL;
ALTER TABLE matches ALTER COLUMN game_start_at SET NOT NULL;

ALTER TABLE matches DROP COLUMN game_scheduled_at;

DROP INDEX IF EXISTS idx_matches_is_public_game_scheduled_master;
DROP INDEX IF EXISTS idx_matches_game_scheduled_at;

CREATE INDEX IF NOT EXISTS idx_matches_is_public_game_start_master ON matches(is_public, game_start_at, master_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_game_start_at ON matches(game_start_at);

COMMIT;
-- +goose StatementEnd
