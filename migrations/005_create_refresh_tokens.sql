-- +goose Up

create table refresh_tokens
(
    id         bigserial primary key,
    user_id    bigint      not null references users (id) on delete cascade,
    token_hash text        not null unique,
    expires_at timestamptz not null,
    revoked_at timestamptz null,
    created_at timestamptz not null default now()
);

create index idx_refresh_tokens_user_id on refresh_tokens (user_id);
create index idx_refresh_tokens_expires_at on refresh_tokens (expires_at);

-- +goose Down
drop table refresh_tokens;