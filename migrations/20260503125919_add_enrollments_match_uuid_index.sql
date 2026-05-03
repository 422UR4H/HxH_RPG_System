-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE INDEX idx_enrollments_match_uuid_created_at
  ON enrollments(match_uuid, created_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_enrollments_match_uuid_created_at;

COMMIT;
-- +goose StatementEnd
