-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE INDEX idx_matches_campaign_uuid_game_scheduled_at
    ON matches(campaign_uuid, game_scheduled_at ASC);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_matches_campaign_uuid_game_scheduled_at;

COMMIT;
-- +goose StatementEnd
