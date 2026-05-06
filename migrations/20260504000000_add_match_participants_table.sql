-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS match_participants (
    id   SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),

    match_uuid           UUID NOT NULL REFERENCES matches(uuid),
    character_sheet_uuid UUID NOT NULL REFERENCES character_sheets(uuid),

    joined_at  TIMESTAMP NOT NULL,
    left_at    TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (uuid),
    UNIQUE (match_uuid, character_sheet_uuid)
);
CREATE INDEX IF NOT EXISTS idx_match_participants_match_uuid ON match_participants(match_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS match_participants;

COMMIT;
-- +goose StatementEnd
