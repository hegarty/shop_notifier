// Package vendor defines the delivery interface every notification
// channel implements, so the platform never couples analytics or the
// consumer loop to one SMS/email/Slack provider. See ADR context in
// shop_docs/docs/architecture.md's notifications section.
package vendor

import "context"

// Sender delivers a rendered message to one destination (a phone number,
// email address, Slack webhook URL, etc. — opaque to this interface).
type Sender interface {
	// Send delivers body to destination. Returns an error if delivery
	// could not be confirmed sent — callers should treat this as
	// retryable, not necessarily "definitely not delivered" (some
	// providers fail after accepting the request).
	Send(ctx context.Context, destination, body string) error
}
