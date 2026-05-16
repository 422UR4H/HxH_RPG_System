-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS turns (
    id          SERIAL PRIMARY KEY,
    uuid        UUID NOT NULL DEFAULT gen_random_uuid(),
    round_uuid  UUID NOT NULL REFERENCES rounds(uuid),
    created_at  TIMESTAMP NOT NULL,
    finished_at TIMESTAMP NOT NULL,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_turns_round_uuid ON turns(round_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS turns;

COMMIT;
-- +goose StatementEnd
