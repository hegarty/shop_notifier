CREATE TABLE notification_channels (
    id             BIGSERIAL PRIMARY KEY,
    tenant_id      TEXT NOT NULL,
    channel_type   TEXT NOT NULL CHECK (channel_type IN ('sms', 'email', 'slack')),
    configuration  JSONB NOT NULL, -- e.g. {"phone_number": "+15551234567"} — never a raw secret; secrets live in Secrets Manager
    enabled        BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_channels_tenant ON notification_channels (tenant_id, enabled);
