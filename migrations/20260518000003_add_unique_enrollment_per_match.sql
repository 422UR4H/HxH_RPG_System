-- +goose Up
CREATE UNIQUE INDEX idx_enrollments_unique_per_match
ON enrollments (match_uuid, character_sheet_uuid);

-- +goose Down
DROP INDEX IF EXISTS idx_enrollments_unique_per_match;
