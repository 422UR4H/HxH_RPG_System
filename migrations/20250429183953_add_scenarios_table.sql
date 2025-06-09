-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS scenarios (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  user_uuid UUID NOT NULL,

  name VARCHAR(32) NOT NULL,
  brief_description VARCHAR(64),
  description TEXT,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  UNIQUE (name),
  FOREIGN KEY (user_uuid) REFERENCES users(uuid)
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS scenarios;

COMMIT;
-- +goose StatementEnd
