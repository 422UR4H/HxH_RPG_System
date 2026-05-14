-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS rounds (
    id         SERIAL PRIMARY KEY,
    uuid       UUID NOT NULL DEFAULT gen_random_uuid(),
    scene_uuid UUID NOT NULL REFERENCES scenes(uuid),
    mode       VARCHAR(16) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    finished_at TIMESTAMP,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_rounds_scene_uuid ON rounds(scene_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS rounds;

COMMIT;
-- +goose StatementEnd
