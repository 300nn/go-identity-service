-- +goose Up

create table outbox_events
(
    id             bigserial primary key,

    event_type     text        not null,
    aggregate_type text        not null,
    aggregate_id   text        not null,

    payload        jsonb       not null,

    status         text        not null default 'NEW',
    attempts       int         not null default 0,
    last_error     text        null,

    locked_at      timestamptz null,
    processed_at   timestamptz null,
    created_at     timestamptz not null default now(),

    constraint outbox_events_status_check
        check (status in ('NEW', 'PROCESSING', 'PROCESSED', 'FAILED'))
);

create index idx_outbox_events_status_created_at
    on outbox_events (status, created_at);

create index idx_outbox_events_aggregate
    on outbox_events (aggregate_type, aggregate_id);

-- +goose Down
drop table outbox_events;