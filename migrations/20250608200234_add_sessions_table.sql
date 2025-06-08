-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    user_uuid UUID NOT NULL,

    token VARCHAR(255) NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_sessions_user_uuid_created_at ON sessions(user_uuid, created_at DESC);
CREATE INDEX idx_sessions_user_uuid_token ON sessions(user_uuid, token);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS sessions;

COMMIT;
-- +goose StatementEnd
