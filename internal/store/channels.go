// Package store looks up a tenant's configured notification channels.
package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Channel is one tenant's configured destination for a channel type (sms,
// email, slack). Configuration is channel-type-specific — {"phone_number":
// "+1..."} for sms, {"webhook_url": "..."} for slack, etc. Never contains a
// secret; provider credentials live in Secrets Manager.
type Channel struct {
	ID            int64
	TenantID      string
	ChannelType   string
	Configuration json.RawMessage
}

type ChannelStore struct {
	pool *pgxpool.Pool
}

func NewChannelStore(pool *pgxpool.Pool) *ChannelStore {
	return &ChannelStore{pool: pool}
}

// EnabledChannels returns every enabled notification channel configured
// for tenantID, across all channel types — the consumer fans out to all of
// them.
func (s *ChannelStore) EnabledChannels(ctx context.Context, tenantID string) ([]Channel, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, channel_type, configuration
		FROM notification_channels
		WHERE tenant_id = $1 AND enabled = true
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("store: query channels for %q: %w", tenantID, err)
	}
	defer rows.Close()

	var channels []Channel
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.TenantID, &c.ChannelType, &c.Configuration); err != nil {
			return nil, fmt.Errorf("store: scan channel: %w", err)
		}
		channels = append(channels, c)
	}
	return channels, rows.Err()
}
