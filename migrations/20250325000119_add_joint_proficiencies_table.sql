-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS joint_proficiencies (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  character_sheet_uuid UUID NOT NULL,

  name VARCHAR(32) NOT NULL,
  weapons TEXT[] NOT NULL,
  exp INT NOT NULL,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  FOREIGN KEY (character_sheet_uuid) REFERENCES character_sheets (uuid) ON DELETE CASCADE
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS joint_proficiencies;

COMMIT;
-- +goose StatementEnd
