-- +goose Up
create table user_audit_events
(
    id bigserial primary key,

    source_event_id text not null unique,
    user_id bigint not null,
    event_type text not null,
    payload jsonb not null,

    created_at timestamptz not null default now()
);

create index idx_user_audit_events_user_id_created_at
    on user_audit_events(user_id, created_at desc);

-- +goose Down
drop table user_audit_events;