-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS submissions (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  campaign_uuid UUID NOT NULL,
  character_sheet_uuid UUID NOT NULL,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  UNIQUE (character_sheet_uuid),
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid),
  FOREIGN KEY (character_sheet_uuid) REFERENCES character_sheets(uuid)
);
CREATE INDEX idx_submissions_campaign_sheet_uuid ON submissions(campaign_uuid, character_sheet_uuid);
CREATE INDEX idx_submissions_character_sheet_uuid ON submissions(character_sheet_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS submissions;

COMMIT;
-- +goose StatementEnd
