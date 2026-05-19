-- +goose Up
-- char_exp is a denormalized copy of the character-level accumulated exp, stored for efficient
-- summary list queries. The authoritative value is always computed by the domain entity
-- (CharacterExp.GetExpPoints) after full sheet reconstruction. Never trust this column for
-- game logic — use the full sheet build instead.
ALTER TABLE character_sheets ADD COLUMN char_exp INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE character_sheets DROP COLUMN char_exp;
