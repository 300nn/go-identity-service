-- +goose Up

DROP TABLE user_events;

-- +goose Down

CREATE TABLE user_events (
                             id BIGSERIAL PRIMARY KEY,
                             user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                             event_type TEXT NOT NULL,
                             payload JSONB NOT NULL DEFAULT '{}'::jsonb,
                             created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_events_user_id ON user_events (user_id);