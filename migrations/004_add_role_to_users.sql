-- +goose Up

ALTER TABLE users
    ADD COLUMN role TEXT NOT NULL DEFAULT 'USER';

ALTER TABLE users
    ADD CONSTRAINT users_role_check
        CHECK (role IN ('USER', 'ADMIN'));

-- +goose Down

ALTER TABLE users
    DROP CONSTRAINT users_role_check;

ALTER TABLE users
    DROP COLUMN role;