-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS enrollments (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  match_uuid UUID NOT NULL,
  character_sheet_uuid UUID NOT NULL,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  UNIQUE (uuid),
  UNIQUE (character_sheet_uuid),
  FOREIGN KEY (match_uuid) REFERENCES matches(uuid),
  FOREIGN KEY (character_sheet_uuid) REFERENCES character_sheets(uuid)
);
CREATE INDEX idx_enrollments_sheet_match_uuid ON enrollments(character_sheet_uuid, match_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS enrollments;

COMMIT;
-- +goose StatementEnd
