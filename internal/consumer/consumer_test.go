package consumer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hegarty/shop_notifier/internal/store"
	"github.com/hegarty/shop_notifier/internal/vendor"
)

type fakeChannels struct {
	channels []store.Channel
}

func (f *fakeChannels) EnabledChannels(_ context.Context, _ string) ([]store.Channel, error) {
	return f.channels, nil
}

type fakeSender struct {
	sent []struct{ destination, body string }
}

func (f *fakeSender) Send(_ context.Context, destination, body string) error {
	f.sent = append(f.sent, struct{ destination, body string }{destination, body})
	return nil
}

func analyticsResultPayload(t *testing.T) json.RawMessage {
	t.Helper()
	payload := map[string]any{
		"tenant_id": "devmoto",
		"job":       "sales.channel.breakdown",
		"period":    map[string]string{"label": "Sep 11"},
		"sales": map[string]any{
			"total":       "100.00",
			"shop":        "100.00",
			"order_count": 1,
			"aov":         "100.00",
			"collective":  map[string]any{"total": "0.00"},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return b
}

func TestHandleRequest_SendsToEnabledSMSChannel(t *testing.T) {
	channels := &fakeChannels{channels: []store.Channel{
		{ID: 1, TenantID: "devmoto", ChannelType: "sms", Configuration: json.RawMessage(`{"phone_number":"+15551234567"}`)},
	}}
	sms := &fakeSender{}
	c := &Consumer{
		Channels:           channels,
		Senders:            map[string]vendor.Sender{"sms": sms},
		TenantDisplayNames: map[string]string{"devmoto": "DevMoto Daily"},
	}

	req := Request{TenantID: "devmoto", Kind: "analytics_result", Job: "sales.channel.breakdown", Payload: analyticsResultPayload(t)}
	if err := c.HandleRequest(t.Context(), req); err != nil {
		t.Fatalf("HandleRequest: %v", err)
	}

	if len(sms.sent) != 1 {
		t.Fatalf("expected 1 SMS sent, got %d", len(sms.sent))
	}
	if sms.sent[0].destination != "+15551234567" {
		t.Errorf("destination = %q, want +15551234567", sms.sent[0].destination)
	}
	if want := "DevMoto Daily — Sep 11"; sms.sent[0].body[:len(want)] != want {
		t.Errorf("body doesn't start with %q: %q", want, sms.sent[0].body)
	}
}

func TestHandleRequest_SkipsChannelTypeWithNoSender(t *testing.T) {
	channels := &fakeChannels{channels: []store.Channel{
		{ID: 1, TenantID: "devmoto", ChannelType: "slack", Configuration: json.RawMessage(`{"webhook_url":"https://example.com"}`)},
	}}
	c := &Consumer{Channels: channels, Senders: map[string]vendor.Sender{}}

	req := Request{TenantID: "devmoto", Kind: "analytics_result", Job: "sales.channel.breakdown", Payload: analyticsResultPayload(t)}
	if err := c.HandleRequest(t.Context(), req); err != nil {
		t.Fatalf("expected no error when no Sender is registered for the channel type, got: %v", err)
	}
}

func TestHandleRequest_UnknownKind(t *testing.T) {
	c := &Consumer{Channels: &fakeChannels{}, Senders: map[string]vendor.Sender{}}
	if err := c.HandleRequest(t.Context(), Request{Kind: "bogus"}); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestHandleRequest_UnknownJob(t *testing.T) {
	c := &Consumer{Channels: &fakeChannels{}, Senders: map[string]vendor.Sender{}}
	req := Request{Kind: "analytics_result", Job: "bogus.job", Payload: json.RawMessage(`{}`)}
	if err := c.HandleRequest(t.Context(), req); err == nil {
		t.Fatal("expected error for unrecognized job")
	}
}
