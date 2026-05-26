-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_campaign_uuid_fkey;
ALTER TABLE matches
  ADD CONSTRAINT matches_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid) ON DELETE CASCADE;

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_campaign_uuid_fkey;
ALTER TABLE submissions
  ADD CONSTRAINT submissions_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid) ON DELETE CASCADE;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_campaign_uuid_fkey;
ALTER TABLE matches
  ADD CONSTRAINT matches_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid);

ALTER TABLE submissions DROP CONSTRAINT IF EXISTS submissions_campaign_uuid_fkey;
ALTER TABLE submissions
  ADD CONSTRAINT submissions_campaign_uuid_fkey
  FOREIGN KEY (campaign_uuid) REFERENCES campaigns(uuid);

COMMIT;
-- +goose StatementEnd
