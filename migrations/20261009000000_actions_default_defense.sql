-- +goose Up
-- +goose StatementBegin
BEGIN;

-- The master's condition on the DEFAULT defense of a reaction (Action.DefaultDefense, Phase 8):
-- the passive defense that stands behind a dodge, closedDodge or escapeGuard. It is its own
-- column, not the defense one, because that one is a component a player declares and the wire
-- carries; this one is the master's, and no surface shows it. NULL means the master did not
-- edit it — every plain action, every other reaction, and every row from before this column.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS default_defense JSONB;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE actions DROP COLUMN IF EXISTS default_defense;

COMMIT;
-- +goose StatementEnd
