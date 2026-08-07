-- +goose Up
create table users (
    id bigserial primary key,
    name text not null,
    email text not null unique,
    age int not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index idx_users_email on users (email);

-- +goose Down
drop table users;
