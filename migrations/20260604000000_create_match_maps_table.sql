-- migrations/20260604000000_create_match_maps_table.sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS match_maps (
  match_uuid  UUID         PRIMARY KEY REFERENCES matches(uuid) ON DELETE CASCADE,
  map_uuid    UUID         NOT NULL    REFERENCES maps(uuid)    ON DELETE RESTRICT,
  attached_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS match_maps;

COMMIT;
-- +goose StatementEnd
