-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS character_profiles (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  character_sheet_uuid UUID NOT NULL,

  nickname VARCHAR(16) NOT NULL,
  fullname VARCHAR(32) NOT NULL,
  alignment VARCHAR(16) NOT NULL,
  character_class VARCHAR(16) NOT NULL,
  long_description TEXT,
  brief_description VARCHAR(32),
  birthday DATE NOT NULL,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  FOREIGN KEY (character_sheet_uuid) REFERENCES character_sheets(uuid) ON DELETE CASCADE
);
CREATE INDEX idx_character_profiles_character_sheet_uuid ON character_profiles(character_sheet_uuid);
CREATE INDEX idx_character_profiles_nickname ON character_profiles(nickname);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS character_profiles;

COMMIT;
-- +goose StatementEnd
