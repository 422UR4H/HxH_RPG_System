-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS campaigns (
  id SERIAL PRIMARY KEY,
  uuid UUID NOT NULL DEFAULT gen_random_uuid(),
  user_uuid UUID NOT NULL,
  scenario_uuid UUID,

  name VARCHAR(32) NOT NULL,
  brief_initial_description VARCHAR(255),
  brief_final_description VARCHAR(255),
  description TEXT,

  is_public BOOLEAN DEFAULT TRUE,
  call_link VARCHAR(255),

  story_start_at DATE NOT NULL,
  story_current_at TIMESTAMP,
  story_end_at DATE,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (uuid),
  FOREIGN KEY (user_uuid) REFERENCES users (uuid),
  FOREIGN KEY (scenario_uuid) REFERENCES scenarios (uuid)
);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS campaigns;

COMMIT;
-- +goose StatementEnd
