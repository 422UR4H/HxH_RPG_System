-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS actions (
    id           SERIAL PRIMARY KEY,
    uuid         UUID NOT NULL DEFAULT gen_random_uuid(),
    turn_uuid    UUID NOT NULL REFERENCES turns(uuid),
    actor_uuid   UUID NOT NULL REFERENCES users(uuid),
    react_to_uuid UUID REFERENCES actions(uuid),
    target_ids   UUID[] NOT NULL DEFAULT '{}',
    type         VARCHAR(32) NOT NULL,
    speed        JSONB,
    skills       JSONB,
    move         JSONB,
    attack       JSONB,
    defense      JSONB,
    dodge        JSONB,
    feint        JSONB,
    trigger      JSONB,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (uuid)
);
CREATE INDEX IF NOT EXISTS idx_actions_turn_uuid ON actions(turn_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS actions;

COMMIT;
-- +goose StatementEnd
