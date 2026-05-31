-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS maps (
  id             SERIAL       PRIMARY KEY,
  uuid           UUID         NOT NULL DEFAULT gen_random_uuid(),
  campaign_uuid  UUID         NOT NULL REFERENCES campaigns(uuid) ON DELETE CASCADE,
  name           VARCHAR(255) NOT NULL,
  description    TEXT         NOT NULL DEFAULT '',
  grid           JSONB        NOT NULL,
  bg             JSONB,
  pieces         JSONB        NOT NULL DEFAULT '[]',
  walls          JSONB        NOT NULL DEFAULT '[]',
  decorations    JSONB        NOT NULL DEFAULT '[]',
  items          JSONB        NOT NULL DEFAULT '[]',
  created_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (uuid)
);

CREATE INDEX IF NOT EXISTS idx_maps_campaign_uuid ON maps(campaign_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_maps_campaign_uuid;
DROP TABLE IF EXISTS maps;

COMMIT;
-- +goose StatementEnd
