-- +goose Up
alter table users
add column password_hash text not null default '';

alter table users
alter column password_hash drop default;


-- +goose Down
alter table users
drop column password_hash;