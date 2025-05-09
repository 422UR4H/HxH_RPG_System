-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS matches (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  master_uuid UUID NOT NULL,
  campaign_uuid UUID NOT NULL,

  title VARCHAR(32) NOT NULL,
  brief_description VARCHAR(64),
  description TEXT,
  story_start_at DATE NOT NULL,
  story_end_at DATE,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  FOREIGN KEY (master_uuid) REFERENCES users (uuid),
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns (uuid)
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS matches;

COMMIT;
-- +goose StatementEnd
