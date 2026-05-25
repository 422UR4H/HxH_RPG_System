-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS enrollments_match_uuid_fkey;
ALTER TABLE enrollments
  ADD CONSTRAINT enrollments_match_uuid_fkey
  FOREIGN KEY (match_uuid) REFERENCES matches(uuid) ON DELETE CASCADE;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS enrollments_match_uuid_fkey;
ALTER TABLE enrollments
  ADD CONSTRAINT enrollments_match_uuid_fkey
  FOREIGN KEY (match_uuid) REFERENCES matches(uuid);

COMMIT;
-- +goose StatementEnd
