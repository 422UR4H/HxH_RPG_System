-- migrations/20261002000000_scenes_rounds_turns_timestamptz.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- Fix D (fix-tz-brief.md, 2026-10-02): scenes/rounds/turns.created_at/finished_at were
-- TIMESTAMP (no time zone). Go wrote the process's local wall clock — a zoned time.Time, e.g.
-- 11:57:33-03:00 — but pgx's TIMESTAMP (no tz) codec drops the zone and keeps the wall-clock
-- digits; reading it back labels those same digits UTC. A turn closed at 11:57 in -03:00 came
-- back as "...11:57:33Z" — three hours before the real instant — while match_events.created_at
-- and master_actions.happened_at (already TIMESTAMPTZ) read back correctly in the SAME history
-- response (get_match_history.go formats every one of them with time.RFC3339).
--
-- No production exists yet: every row on disk was written by this dev host, whose zone is
-- America/Belem (-03:00, no DST — the offset never shifts under a backfill). AT TIME ZONE
-- interprets the stored wall-clock digits as having been written in that zone and converts
-- them to the correct absolute instant. Writers need no change: a Go time.Time is always
-- zoned, and TIMESTAMPTZ's codec already encodes/decodes it as the correct instant regardless
-- of which zone it was captured in (time.Now() is enough — see gateway-conventions
-- "Timestamps: Go, Not SQL").
ALTER TABLE scenes ALTER COLUMN created_at  TYPE TIMESTAMPTZ USING created_at  AT TIME ZONE 'America/Belem';
ALTER TABLE scenes ALTER COLUMN finished_at TYPE TIMESTAMPTZ USING finished_at AT TIME ZONE 'America/Belem';
ALTER TABLE rounds ALTER COLUMN created_at  TYPE TIMESTAMPTZ USING created_at  AT TIME ZONE 'America/Belem';
ALTER TABLE rounds ALTER COLUMN finished_at TYPE TIMESTAMPTZ USING finished_at AT TIME ZONE 'America/Belem';
ALTER TABLE turns  ALTER COLUMN created_at  TYPE TIMESTAMPTZ USING created_at  AT TIME ZONE 'America/Belem';
ALTER TABLE turns  ALTER COLUMN finished_at TYPE TIMESTAMPTZ USING finished_at AT TIME ZONE 'America/Belem';

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

-- Symmetric: AT TIME ZONE on a TIMESTAMPTZ returns the instant's wall clock IN that zone,
-- exactly undoing the Up conversion back to the original on-disk digits.
ALTER TABLE scenes ALTER COLUMN created_at  TYPE TIMESTAMP USING created_at  AT TIME ZONE 'America/Belem';
ALTER TABLE scenes ALTER COLUMN finished_at TYPE TIMESTAMP USING finished_at AT TIME ZONE 'America/Belem';
ALTER TABLE rounds ALTER COLUMN created_at  TYPE TIMESTAMP USING created_at  AT TIME ZONE 'America/Belem';
ALTER TABLE rounds ALTER COLUMN finished_at TYPE TIMESTAMP USING finished_at AT TIME ZONE 'America/Belem';
ALTER TABLE turns  ALTER COLUMN created_at  TYPE TIMESTAMP USING created_at  AT TIME ZONE 'America/Belem';
ALTER TABLE turns  ALTER COLUMN finished_at TYPE TIMESTAMP USING finished_at AT TIME ZONE 'America/Belem';

COMMIT;
-- +goose StatementEnd
