-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS submit_character_sheets (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  campaign_uuid UUID NOT NULL,
  character_sheet_uuid UUID NOT NULL,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  UNIQUE (character_sheet_uuid),
  FOREIGN KEY (character_sheet_uuid) REFERENCES character_sheets (uuid),
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns (uuid)
);
CREATE INDEX idx_submit_character_sheet_campaign_uuid ON submit_character_sheets (campaign_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS submit_character_sheets;

COMMIT;
-- +goose StatementEnd
