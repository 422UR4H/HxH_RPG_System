-- +goose Up
-- +goose StatementBegin
BEGIN;

-- The declaration of a turn was persisted; the COLLISION never was. Margin, damage, ladder
-- rung and chain state existed only in memory, and recomputing them afterwards is impossible:
-- the ModifierLedger of that instant is gone with the process.
--
-- JSONB rather than tables because this is the snapshot of a calculation, not a queryable
-- entity — nobody is going to ask "every turn with damage over 10" in the MVP — and because
-- it is one more write in a transaction that already exists.
--
-- The resolution written here is the SETTLED one, the one whose damage was applied. The
-- master's edits recompute the projection many times; only the close is persisted.
ALTER TABLE turns ADD COLUMN IF NOT EXISTS resolution JSONB;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE turns DROP COLUMN IF EXISTS resolution;

COMMIT;
-- +goose StatementEnd
