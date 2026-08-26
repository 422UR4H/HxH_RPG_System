-- +goose Up
-- +goose StatementBegin
BEGIN;

-- What a master's edit DISPLACED — never the edit itself. The action already carries the new
-- value; storing both would be duplication that diverges, and the original is reconstructed by
-- reading this row against the corresponding row of actions.
--
-- The name refuses things, which is the whole reason it is not called SystemData: a generic
-- name cannot turn anything away and becomes a dumping ground. Wall state cannot be put in
-- overridden_action_values without the name being visibly false.
--
-- ONE ROW PER FIELD, holding the ORIGINAL — hence the unique constraint. Not one row per edit:
-- the master's intermediate values are neither what the player sent nor what the system
-- calculated, and those two are the only things this table exists to preserve.
--
-- No source column. If only the master overrides, there is nothing for it to discriminate; the
-- bias the SYSTEM applies is a Modifier in the ledger and was never an override. `origin` is a
-- different question: where the DISPLACED value came from.
--
-- Composition rule: original_value is a snapshot taken at the moment of ITS OWN row's edit,
-- not a pristine copy of what the player originally sent. Edit a skill's condition, then
-- remove that skill entirely, and the "skills" row's original_value for it already carries
-- the earlier condition baked in — origin still reads 'player' regardless, since origin names
-- where the displaced VALUE's shape came from, not the edit history behind it. The player's
-- true original is recoverable by composing that entry with the "skill:<name>.condition"
-- row's own original_value, not by reading either row alone. See the full rule on
-- match.OverriddenValue (internal/domain/match/override.go).
CREATE TABLE IF NOT EXISTS overridden_action_values (
    id             SERIAL PRIMARY KEY,
    action_uuid    UUID NOT NULL REFERENCES actions(uuid) ON DELETE CASCADE,
    field          VARCHAR(64) NOT NULL,
    origin         VARCHAR(16) NOT NULL,
    master_uuid    UUID NOT NULL REFERENCES users(uuid),
    overridden_at  TIMESTAMP NOT NULL,
    original_value JSONB,
    -- UNIQUE (action_uuid, field) already creates a btree index led by action_uuid, which
    -- covers WHERE action_uuid = $1 lookups on its own — a separate single-column index on
    -- action_uuid would be a redundant duplicate, not a distinct access path.
    UNIQUE (action_uuid, field)
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS overridden_action_values;

COMMIT;
-- +goose StatementEnd
