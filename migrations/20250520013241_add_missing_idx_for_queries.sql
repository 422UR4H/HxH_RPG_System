-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE INDEX IF NOT EXISTS idx_character_sheets_player_uuid ON character_sheets(player_uuid);

CREATE INDEX IF NOT EXISTS idx_character_profiles_character_sheet_uuid ON character_profiles(character_sheet_uuid);
CREATE INDEX IF NOT EXISTS idx_character_profiles_nickname ON character_profiles(nickname);

CREATE INDEX IF NOT EXISTS idx_proficiencies_character_sheet_uuid ON proficiencies(character_sheet_uuid);
CREATE INDEX IF NOT EXISTS idx_joint_proficiencies_character_sheet_uuid ON joint_proficiencies(character_sheet_uuid);

CREATE INDEX IF NOT EXISTS idx_campaigns_master_uuid ON campaigns(master_uuid);
CREATE INDEX IF NOT EXISTS idx_campaigns_master_uuid_name ON campaigns(master_uuid, name);
CREATE INDEX IF NOT EXISTS idx_campaigns_scenario_uuid ON campaigns(scenario_uuid);

CREATE INDEX IF NOT EXISTS idx_scenarios_user_uuid ON scenarios(user_uuid);
CREATE INDEX IF NOT EXISTS idx_scenarios_user_uuid_name ON scenarios(user_uuid, name);
CREATE INDEX IF NOT EXISTS idx_scenarios_name ON scenarios(name);

CREATE INDEX IF NOT EXISTS idx_matches_master_uuid ON matches(master_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_master_uuid_title ON matches(master_uuid, title);
CREATE INDEX IF NOT EXISTS idx_matches_campaign_uuid ON matches(campaign_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_game_start_at ON matches(game_start_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_character_sheets_player_uuid;
DROP INDEX IF EXISTS idx_character_profiles_character_sheet_uuid;
DROP INDEX IF EXISTS idx_character_profiles_nickname;
DROP INDEX IF EXISTS idx_proficiencies_character_sheet_uuid;
DROP INDEX IF EXISTS idx_joint_proficiencies_character_sheet_uuid;

DROP INDEX IF EXISTS idx_campaigns_master_uuid;
DROP INDEX IF EXISTS idx_campaigns_master_uuid_name;
DROP INDEX IF EXISTS idx_campaigns_scenario_uuid;

DROP INDEX IF EXISTS idx_scenarios_user_uuid;
DROP INDEX IF EXISTS idx_scenarios_user_uuid_name;
DROP INDEX IF EXISTS idx_scenarios_name;

DROP INDEX IF EXISTS idx_matches_master_uuid;
DROP INDEX IF EXISTS idx_matches_master_uuid_title;
DROP INDEX IF EXISTS idx_matches_campaign_uuid;

COMMIT;
-- +goose StatementEnd