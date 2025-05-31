-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE character_sheets
ADD COLUMN player_uuid UUID NULL,
ADD COLUMN master_uuid UUID NULL,
ADD COLUMN campaign_uuid UUID NULL;

CREATE INDEX idx_character_sheets_player_uuid ON character_sheets (player_uuid);
CREATE INDEX idx_character_sheets_master_uuid ON character_sheets (master_uuid);
CREATE INDEX idx_character_sheets_campaign_uuid ON character_sheets (campaign_uuid);

-- Add CHECK constraint to ensure uniqueness (logical XOR)
ALTER TABLE character_sheets
ADD CONSTRAINT chk_exclusive_owner 
CHECK ((player_uuid IS NULL AND master_uuid IS NOT NULL) OR 
       (player_uuid IS NOT NULL AND master_uuid IS NULL));

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE character_sheets DROP CONSTRAINT IF EXISTS chk_exclusive_owner;

DROP INDEX IF EXISTS idx_character_sheets_player_uuid;
DROP INDEX IF EXISTS idx_character_sheets_master_uuid;
DROP INDEX IF EXISTS idx_character_sheets_campaign_uuid;

ALTER TABLE character_sheets DROP COLUMN IF EXISTS player_uuid;
ALTER TABLE character_sheets DROP COLUMN IF EXISTS master_uuid;
ALTER TABLE character_sheets DROP COLUMN IF EXISTS campaign_uuid;

COMMIT;
-- +goose StatementEnd