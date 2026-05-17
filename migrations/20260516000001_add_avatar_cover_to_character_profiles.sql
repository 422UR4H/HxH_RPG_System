-- +goose Up
-- +goose StatementBegin
ALTER TABLE character_profiles
  ADD COLUMN avatar_url TEXT,
  ADD COLUMN cover_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE character_profiles
  DROP COLUMN IF EXISTS avatar_url,
  DROP COLUMN IF EXISTS cover_url;
-- +goose StatementEnd
