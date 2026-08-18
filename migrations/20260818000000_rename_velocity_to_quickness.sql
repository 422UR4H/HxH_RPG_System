-- migrations/20260818000000_rename_velocity_to_quickness.sql
-- Renames the Agility skill that was called "Velocity" to "Quickness".
--
-- The name was wrong twice over. Under Agility the three skills are Accelerate, Brake and
-- Quickness — starting a movement, stopping one, and the footwork that moves a character
-- inside a slot. "Velocity" named the third after a quantity rather than after what it
-- tests, and it collided with action.Velocity, the movement VECTOR, which is a different
-- thing entirely and stays as it is.
--
-- The column holds experience points positionally, so this is a pure rename with no data
-- migration.
-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE character_sheets RENAME COLUMN velocity_exp TO quickness_exp;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE character_sheets RENAME COLUMN quickness_exp TO velocity_exp;

COMMIT;
-- +goose StatementEnd
