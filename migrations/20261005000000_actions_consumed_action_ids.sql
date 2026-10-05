-- +goose Up
-- +goose StatementBegin
BEGIN;

-- What a charged REACTION took off the reactor's queue when it was attached — one pending
-- action per bar its kind charges, the best-keyed one (Phase 7, item 3). The owner reconciles
-- what they had declared against it: an action named here was CONSUMED, not lost, and a
-- reconnect after the turn closed — even after a restart — can only learn that here.
-- NULL means nothing consumed: a free reaction, a charged one that found nothing queued, every
-- plain action, and every row from before this column.
ALTER TABLE actions ADD COLUMN IF NOT EXISTS consumed_action_ids UUID[];

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER TABLE actions DROP COLUMN IF EXISTS consumed_action_ids;

COMMIT;
-- +goose StatementEnd
