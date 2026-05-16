-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS scenes (
    id          SERIAL PRIMARY KEY,
    uuid        UUID NOT NULL DEFAULT gen_random_uuid(),
    match_uuid  UUID NOT NULL REFERENCES matches(uuid),
    category    VARCHAR(32) NOT NULL,
    brief_initial_description VARCHAR(255) NOT NULL DEFAULT '',
    brief_final_description   VARCHAR(255),
    created_at  TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_scenes_match_uuid ON scenes(match_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS scenes;

COMMIT;
-- +goose StatementEnd
