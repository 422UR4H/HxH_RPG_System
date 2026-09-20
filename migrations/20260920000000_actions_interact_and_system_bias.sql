-- +goose Up
-- +goose StatementBegin
BEGIN;

-- Two pieces of an Action the table had no column for.
--
-- interact: Action.Interact has existed since the wall/door work, and deriveActionType did
-- not even know the field. An interaction carries no other payload — opening a door sets
-- Interact and nothing else — so the whole action persisted as type 'unspecified' with every
-- JSONB column NULL: a row that says something happened and refuses to say what.
--
-- system_bias: the engine-imposed advantage/disadvantage the action was last derived under
-- (0 for a plain action, -1 for a reaction that displaced a queued one). It is a third origin
-- alongside the master's RollCondition and the character's ModifierLedger, and without it the
-- history keeps the number while losing the reason it was that number — on a surface whose
-- entire purpose is letting the table reconstruct the reasoning.
--
-- NOT NULL DEFAULT 0 on system_bias: 0 is the real value for every action that was never
-- biased, which is nearly all of them, and it is exactly what the zero value of the Go field
-- already means. A nullable column would invent a third state ("unknown") that the domain
-- has no way to express.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS interact JSONB;
ALTER TABLE actions ADD COLUMN IF NOT EXISTS system_bias INTEGER NOT NULL DEFAULT 0;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE actions DROP COLUMN IF EXISTS system_bias;
ALTER TABLE actions DROP COLUMN IF EXISTS interact;

COMMIT;
-- +goose StatementEnd
