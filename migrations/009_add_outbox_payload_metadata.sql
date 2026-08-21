-- +goose Up
ALTER TABLE outbox_events
    ADD COLUMN payload_bytes BYTEA,
    ADD COLUMN content_type  TEXT NOT NULL DEFAULT 'application/json',
    ADD COLUMN proto_message TEXT NOT NULL DEFAULT '',
    ADD COLUMN event_version TEXT NOT NULL DEFAULT 'v1';

UPDATE outbox_events
SET payload_bytes = convert_to(payload::text, 'UTF8')
WHERE payload_bytes IS NULL;

ALTER TABLE outbox_events
    ALTER COLUMN payload DROP NOT NULL,
    ALTER COLUMN payload_bytes SET NOT NULL;

ALTER TABLE outbox_events
    ADD CONSTRAINT outbox_events_content_type_check
        CHECK (content_type IN ('application/json', 'application/x-protobuf')),
    ADD CONSTRAINT outbox_events_proto_message_check
        CHECK (
            content_type <> 'application/x-protobuf'
                OR proto_message <> ''
            );

-- +goose Down
ALTER TABLE outbox_events
    DROP CONSTRAINT outbox_events_proto_message_check,
    DROP CONSTRAINT outbox_events_content_type_check;

UPDATE outbox_events
SET payload = '{}'::jsonb
WHERE payload IS NULL;

ALTER TABLE outbox_events
    ALTER COLUMN payload SET NOT NULL;

ALTER TABLE outbox_events
    DROP COLUMN event_version,
    DROP COLUMN proto_message,
    DROP COLUMN content_type,
    DROP COLUMN payload_bytes;