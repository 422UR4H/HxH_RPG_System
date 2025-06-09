-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS proficiencies (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  character_sheet_uuid UUID NOT NULL,

  weapon VARCHAR(16) NOT NULL,
  exp INT NOT NULL,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  FOREIGN KEY (character_sheet_uuid) REFERENCES character_sheets(uuid) ON DELETE CASCADE
);
CREATE INDEX idx_proficiencies_character_sheet_uuid ON proficiencies(character_sheet_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS proficiencies;

COMMIT;
-- +goose StatementEnd
