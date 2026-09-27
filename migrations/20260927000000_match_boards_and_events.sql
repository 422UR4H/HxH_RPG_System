-- migrations/20260927000000_match_boards_and_events.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- The match's own board: a SNAPSHOT that starts as a copy of the campaign map it was attached
-- to and then belongs to the match alone. Editing the campaign map never reaches it, and two
-- matches on one map never touch each other's pieces (front-combat-phases.md §6A.5, B3).
-- bg NULL means "inherit the map's background"; the future in-match map editor writes here.
CREATE TABLE IF NOT EXISTS match_boards (
  match_uuid  UUID        PRIMARY KEY REFERENCES matches(uuid) ON DELETE CASCADE,
  map_uuid    UUID        NOT NULL    REFERENCES maps(uuid)    ON DELETE RESTRICT,
  grid        JSONB       NOT NULL,
  bg          JSONB,
  pieces      JSONB       NOT NULL DEFAULT '[]',
  walls       JSONB       NOT NULL DEFAULT '[]',
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- What happens INSIDE a round, is not a turn and is nobody's action (B15): a regime change.
CREATE TABLE IF NOT EXISTS match_events (
  uuid        UUID        PRIMARY KEY,
  match_uuid  UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  scene_uuid  UUID        NOT NULL REFERENCES scenes(uuid),
  round_uuid  UUID        NOT NULL REFERENCES rounds(uuid),
  kind        VARCHAR(32) NOT NULL,
  payload     JSONB       NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_match_events_match ON match_events(match_uuid, created_at);

-- The master's own actions, in a table of their own and NOT in actions — a decision of the
-- product owner, and a matter of the model rather than taste (front-combat-phases.md §6A.5,
-- B14): actions.actor_uuid references character_sheets while a master action's actor is the
-- master, a USER; and actions.turn_uuid is NOT NULL while a master action happens outside any
-- turn (dragging between turns is the common case). Mixing them would loosen both columns and
-- force every history read to filter one kind out of the other.
--
-- turn_uuid is NULL outside a turn, and has no FK on purpose: a turn is only written when it
-- closes, and a restart can lose it after the master action was already recorded — which the
-- history then shows outside any turn. views is what each player saw of it LIVE
-- ({"<playerUUID>": "full"|"left"}; absent = saw nothing), so the history can show every
-- reader exactly that and no more. edit_action is NOT a master action and never lands here.
CREATE TABLE IF NOT EXISTS master_actions (
  uuid         UUID        PRIMARY KEY,
  match_uuid   UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  scene_uuid   UUID        NOT NULL REFERENCES scenes(uuid),
  round_uuid   UUID        NOT NULL REFERENCES rounds(uuid),
  turn_uuid    UUID,
  master_uuid  UUID        NOT NULL REFERENCES users(uuid),
  kind         VARCHAR(32) NOT NULL,
  content      JSONB       NOT NULL,
  views        JSONB       NOT NULL DEFAULT '{}',
  happened_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_master_actions_match ON master_actions(match_uuid, happened_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;
DROP INDEX IF EXISTS idx_master_actions_match;
DROP TABLE IF EXISTS master_actions;
DROP INDEX IF EXISTS idx_match_events_match;
DROP TABLE IF EXISTS match_events;
DROP TABLE IF EXISTS match_boards;
COMMIT;
-- +goose StatementEnd
