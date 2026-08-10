-- +goose Up
CREATE TABLE user_profiles (
                               id BIGSERIAL PRIMARY KEY,
                               user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                               bio TEXT NOT NULL DEFAULT '',
                               created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                               updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_user_profiles_user_id ON user_profiles (user_id);

CREATE TABLE user_events (
                             id BIGSERIAL PRIMARY KEY,
                             user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                             event_type TEXT NOT NULL,
                             payload JSONB NOT NULL DEFAULT '{}'::jsonb,
                             created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_events_user_id ON user_events (user_id);

-- +goose Down
DROP TABLE user_events;
DROP TABLE user_profiles;