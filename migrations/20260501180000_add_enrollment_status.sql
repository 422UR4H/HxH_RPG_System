-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE enrollments ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';

ALTER TABLE enrollments DROP CONSTRAINT enrollments_character_sheet_uuid_key;

CREATE UNIQUE INDEX idx_enrollments_active_sheet
ON enrollments (character_sheet_uuid)
WHERE status != 'rejected';

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_enrollments_active_sheet;

ALTER TABLE enrollments ADD CONSTRAINT enrollments_character_sheet_uuid_key UNIQUE (character_sheet_uuid);

ALTER TABLE enrollments DROP COLUMN status;

COMMIT;
-- +goose StatementEnd
