-- migrations/20261108000000_story_clock.sql
-- +goose Up
-- +goose StatementBegin

-- Relógio da história: horário inicial da partida, encerramento real e o estado do relógio.
ALTER TABLE matches ADD COLUMN IF NOT EXISTS story_start_time TIME NULL;
ALTER TABLE matches ADD COLUMN IF NOT EXISTS game_end_at TIMESTAMPTZ NULL;

CREATE TABLE IF NOT EXISTS match_clocks (
    match_uuid    UUID PRIMARY KEY REFERENCES matches(uuid) ON DELETE CASCADE,
    story_at      TIMESTAMP NULL,
    limit_at      TIMESTAMP NULL,
    limit_reason  VARCHAR(16) NOT NULL DEFAULT '',
    running_since TIMESTAMPTZ NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE scenes ADD COLUMN IF NOT EXISTS story_started_at TIMESTAMP NULL;
ALTER TABLE scenes ADD COLUMN IF NOT EXISTS story_finished_at TIMESTAMP NULL;

ALTER TABLE rounds ADD COLUMN IF NOT EXISTS counted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE rounds ADD COLUMN IF NOT EXISTS drained BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE turns ADD COLUMN IF NOT EXISTS story_at TIMESTAMP NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE turns DROP COLUMN IF EXISTS story_at;

ALTER TABLE rounds DROP COLUMN IF EXISTS drained;
ALTER TABLE rounds DROP COLUMN IF EXISTS counted;

ALTER TABLE scenes DROP COLUMN IF EXISTS story_finished_at;
ALTER TABLE scenes DROP COLUMN IF EXISTS story_started_at;

DROP TABLE IF EXISTS match_clocks;

ALTER TABLE matches DROP COLUMN IF EXISTS game_end_at;
ALTER TABLE matches DROP COLUMN IF EXISTS story_start_time;

-- +goose StatementEnd
