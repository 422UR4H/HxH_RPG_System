-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_match_participants_character_sheet_uuid
    ON match_participants(character_sheet_uuid);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_match_participants_character_sheet_uuid;
-- +goose StatementEnd
