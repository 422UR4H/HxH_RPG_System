-- migrations/20261010000000_cancelled_actions.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- An action its declarer took back out of the queue before it opened (cancel_action, spec
-- 2026-10-10 §8.1 P6). A table of its own and NOT actions: actions.turn_uuid is NOT NULL, and a
-- cancelled action never had a turn — the same reason master_actions has its own.
--
-- Written like a master action (P7): with no turn open, at the instant of the cancellation;
-- with one open, together with that turn's close, in its transaction — so turn_uuid (the turn
-- open at the instant, NULL when none was) has no FK, as in master_actions. uuid is the
-- cancelled action's own id.
--
-- action is the whole action in the domain's STORAGE format, the same JSON pieces insertAction
-- writes to the columns of actions (internal/gateway/pg/actioncodec), as one object keyed by
-- those column names: "speed" (always), "skills", "move", "attack", "defense", "dodge",
-- "repel", "feint", "trigger", "interact", "default_defense" (each absent when the action has
-- no such component), "target_ids" (always, [] when none), "system_bias" (always),
-- "react_to_uuid", "reaction_kind", "consumed_action_ids" (absent unless set).
CREATE TABLE IF NOT EXISTS cancelled_actions (
  uuid          UUID        PRIMARY KEY,
  match_uuid    UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  scene_uuid    UUID        NOT NULL REFERENCES scenes(uuid),
  round_uuid    UUID        NOT NULL REFERENCES rounds(uuid),
  turn_uuid     UUID,
  actor_uuid    UUID        NOT NULL REFERENCES character_sheets(uuid),
  cancelled_by  UUID        NOT NULL REFERENCES users(uuid),
  action        JSONB       NOT NULL,
  cancelled_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cancelled_actions_match ON cancelled_actions(match_uuid, cancelled_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;
DROP INDEX IF EXISTS idx_cancelled_actions_match;
DROP TABLE IF EXISTS cancelled_actions;
COMMIT;
-- +goose StatementEnd
