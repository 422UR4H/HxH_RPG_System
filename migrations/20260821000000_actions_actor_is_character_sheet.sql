-- +goose Up
-- +goose StatementBegin
BEGIN;

-- Action.actorID has been the CHARACTER SHEET uuid since phase 2, not the player's. The FK
-- kept pointing at users, so every PersistTurnClose in a real match failed with 23503 —
-- and room.go logs that error and carries on, which is why it went unnoticed: nothing has
-- been persisted from a live match since.
--
-- NOT VALID skips the check against existing rows. Any row already in this table was written
-- before phase 2 and is unreadable anyway (there is no read path yet); the constraint is
-- enforced from here on, which is the part that matters.
ALTER TABLE actions DROP CONSTRAINT IF EXISTS actions_actor_uuid_fkey;
ALTER TABLE actions ADD CONSTRAINT actions_actor_uuid_fkey
    FOREIGN KEY (actor_uuid) REFERENCES character_sheets(uuid) NOT VALID;

-- Phase 4 gave reactions a declared kind and gave the repel its own component. Neither had
-- anywhere to land, and reactions were never written at all.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS reaction_kind VARCHAR(32);
ALTER TABLE actions ADD COLUMN IF NOT EXISTS repel JSONB;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE actions DROP COLUMN IF EXISTS repel;
ALTER TABLE actions DROP COLUMN IF EXISTS reaction_kind;

ALTER TABLE actions DROP CONSTRAINT IF EXISTS actions_actor_uuid_fkey;
ALTER TABLE actions ADD CONSTRAINT actions_actor_uuid_fkey
    FOREIGN KEY (actor_uuid) REFERENCES users(uuid) NOT VALID;

COMMIT;
-- +goose StatementEnd
