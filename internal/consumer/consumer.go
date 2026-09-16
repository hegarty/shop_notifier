// Package consumer wires notifications.requested messages to the right
// formatter and Sender for each of a tenant's configured channels.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hegarty/shop_notifier/internal/format"
	"github.com/hegarty/shop_notifier/internal/store"
	"github.com/hegarty/shop_notifier/internal/vendor"
)

// Request mirrors shop_analytics's worker.NotificationRequest wire shape.
// Duplicated deliberately — see format.AnalyticsResultPayload's doc
// comment for why this package doesn't import shop_analytics.
type Request struct {
	TenantID string          `json:"tenant_id"`
	Kind     string          `json:"kind"`
	Job      string          `json:"job"`
	Payload  json.RawMessage `json:"payload"`
}

// ChannelLookup resolves a tenant's enabled notification channels.
type ChannelLookup interface {
	EnabledChannels(ctx context.Context, tenantID string) ([]store.Channel, error)
}

// Consumer formats and delivers one notification request to every enabled
// channel a tenant has configured.
type Consumer struct {
	Channels ChannelLookup
	// Senders maps channel_type ("sms", "email", "slack") to the Sender
	// that delivers it. A channel type with no registered Sender is
	// skipped (logged, not an error) rather than failing the whole
	// request — see HandleRequest.
	Senders map[string]vendor.Sender
	// TenantDisplayNames provides a friendlier name than the raw tenant ID
	// for message headers (e.g. "devmoto" -> "DevMoto Daily"). Falls back
	// to the bare tenant ID if absent. MVP: static config, not a DB table
	// — see README.md.
	TenantDisplayNames map[string]string
}

func (c *Consumer) HandleRequest(ctx context.Context, req Request) error {
	switch req.Kind {
	case "analytics_result":
		return c.handleAnalyticsResult(ctx, req)
	default:
		return fmt.Errorf("consumer: unknown notification kind %q", req.Kind)
	}
}

func (c *Consumer) handleAnalyticsResult(ctx context.Context, req Request) error {
	var payload format.AnalyticsResultPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return fmt.Errorf("consumer: decode analytics result payload: %w", err)
	}

	var body string
	switch req.Job {
	case "sales.channel.breakdown":
		msg, err := format.SalesChannelBreakdown(c.displayName(req.TenantID), payload)
		if err != nil {
			return fmt.Errorf("consumer: format sales.channel.breakdown: %w", err)
		}
		body = msg
	default:
		return fmt.Errorf("consumer: no formatter registered for job %q", req.Job)
	}

	channels, err := c.Channels.EnabledChannels(ctx, req.TenantID)
	if err != nil {
		return fmt.Errorf("consumer: load channels for %q: %w", req.TenantID, err)
	}

	var lastErr error
	for _, ch := range channels {
		sender, ok := c.Senders[ch.ChannelType]
		if !ok {
			continue // no Sender configured for this channel type yet
		}
		destination, err := destinationFor(ch)
		if err != nil {
			lastErr = err
			continue
		}
		if err := sender.Send(ctx, destination, body); err != nil {
			lastErr = fmt.Errorf("consumer: send via %s to channel %d: %w", ch.ChannelType, ch.ID, err)
		}
	}
	return lastErr
}

func (c *Consumer) displayName(tenantID string) string {
	if name, ok := c.TenantDisplayNames[tenantID]; ok {
		return name
	}
	return tenantID
}

func destinationFor(ch store.Channel) (string, error) {
	switch ch.ChannelType {
	case "sms":
		var cfg struct {
			PhoneNumber string `json:"phone_number"`
		}
		if err := json.Unmarshal(ch.Configuration, &cfg); err != nil {
			return "", fmt.Errorf("consumer: decode sms configuration for channel %d: %w", ch.ID, err)
		}
		return cfg.PhoneNumber, nil
	case "slack":
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal(ch.Configuration, &cfg); err != nil {
			return "", fmt.Errorf("consumer: decode slack configuration for channel %d: %w", ch.ID, err)
		}
		return cfg.WebhookURL, nil
	case "email":
		var cfg struct {
			Address string `json:"address"`
		}
		if err := json.Unmarshal(ch.Configuration, &cfg); err != nil {
			return "", fmt.Errorf("consumer: decode email configuration for channel %d: %w", ch.ID, err)
		}
		return cfg.Address, nil
	default:
		return "", fmt.Errorf("consumer: unknown channel type %q", ch.ChannelType)
	}
}
