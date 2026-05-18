-- +goose Up
DROP INDEX IF EXISTS idx_enrollments_active_sheet;

-- +goose Down
CREATE UNIQUE INDEX idx_enrollments_active_sheet
ON enrollments (character_sheet_uuid)
WHERE status != 'rejected';
