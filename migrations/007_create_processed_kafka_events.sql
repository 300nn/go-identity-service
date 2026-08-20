-- +goose Up
create table processed_kafka_events(
    event_id text primary key,
    topic text not null,
    partition int not null,
    offset_value bigint not null,
    event_type text not null,
    processed_at timestamptz not null default now()
);

create index idx_processed_kafka_events_processed_at
    on processed_kafka_events(processed_at);

-- +goose Down
drop table processed_kafka_events;