-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE character_sheets
ADD COLUMN player_id UUID NULL,
ADD COLUMN scenario_id UUID NULL;

CREATE INDEX idx_character_sheets_player_id ON character_sheets (player_id);
CREATE INDEX idx_character_sheets_scenario_id ON character_sheets (scenario_id);

-- Add CHECK constraint to ensure uniqueness (logical XOR)
ALTER TABLE character_sheets
ADD CONSTRAINT chk_exclusive_owner 
CHECK ((player_id IS NULL AND scenario_id IS NOT NULL) OR 
       (player_id IS NOT NULL AND scenario_id IS NULL));

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE character_sheets DROP CONSTRAINT IF EXISTS chk_exclusive_owner;

DROP INDEX IF EXISTS idx_character_sheets_player_id;
DROP INDEX IF EXISTS idx_character_sheets_scenario_id;

ALTER TABLE character_sheets DROP COLUMN IF EXISTS player_id;
ALTER TABLE character_sheets DROP COLUMN IF EXISTS scenario_id;

COMMIT;
-- +goose StatementEnd