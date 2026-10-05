-- +goose Up
-- +goose StatementBegin
BEGIN;

-- What each player of the session saw, LIVE, of the action's move (owner decision,
-- 2026-10-01): GET /history shows each reader a move only as they saw it at the table. The fog
-- at read time is not the fog of that moment, so the history cannot recompute it — the verdict
-- the live gate reached when the turn opened is recorded here instead, the same way
-- master_actions.views records it for a master action.
--
-- {"<playerUUID>": "full" | "left"}: full = from and position, left = from only, a player with
-- no key saw neither. The master and the actor's owner are never in it (they always see it
-- all). NULL means not recorded: an action with no move, or a row from before this column —
-- the history reads both as "nobody saw it" and fails closed (category only).
--
-- Only on the turn's action. A reaction's displacement (an escape) is decided at the close, so
-- its landing's verdicts live in turns.resolution, inside the escape entry (landingViews).
ALTER TABLE actions ADD COLUMN IF NOT EXISTS move_views JSONB;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE actions DROP COLUMN IF EXISTS move_views;

COMMIT;
-- +goose StatementEnd
