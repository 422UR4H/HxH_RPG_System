-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE INDEX IF NOT EXISTS idx_campaigns_master_uuid_name ON campaigns(master_uuid, name);
CREATE INDEX IF NOT EXISTS idx_campaigns_scenario_uuid ON campaigns(scenario_uuid);

CREATE INDEX IF NOT EXISTS idx_scenarios_user_uuid_name ON scenarios(user_uuid, name);
CREATE INDEX IF NOT EXISTS idx_scenarios_name ON scenarios(name);

CREATE INDEX IF NOT EXISTS idx_matches_is_public_game_start_master ON matches(is_public, game_start_at, master_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_master_uuid_story_start ON matches(master_uuid, story_start_at);
CREATE INDEX IF NOT EXISTS idx_matches_master_uuid_title ON matches(master_uuid, title);
CREATE INDEX IF NOT EXISTS idx_matches_campaign_uuid ON matches(campaign_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_game_start_at ON matches(game_start_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_campaigns_master_uuid_name;
DROP INDEX IF EXISTS idx_campaigns_scenario_uuid;

DROP INDEX IF EXISTS idx_scenarios_user_uuid_name;
DROP INDEX IF EXISTS idx_scenarios_name;

DROP INDEX IF EXISTS idx_matches_is_public_game_start_master;
DROP INDEX IF EXISTS idx_matches_master_uuid_story_start;
DROP INDEX IF EXISTS idx_matches_master_uuid_title;
DROP INDEX IF EXISTS idx_matches_campaign_uuid;
DROP INDEX IF EXISTS idx_matches_game_start_at;

COMMIT;
-- +goose StatementEnd
