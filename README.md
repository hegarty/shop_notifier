# shop_notifier

Notification delivery for the commerce-intel platform: consumes `notifications.requested`
(published by `shop_analytics`, never producing SMS/email/Slack traffic from analytics
code directly), formats a message, and delivers it to each of a tenant's configured
channels. See [shop_docs](https://github.com/hegarty/shop_docs) for the architecture.

## What's here

- **`cmd/notifier`** — consumes `notifications.requested`, formats, delivers.
- **`cmd/migrate`** — applies/rolls back this repo's schema migrations
  (`notification_channels`).
- **`internal/format`** — renders analytics results as text.
  `SalesChannelBreakdown` reproduces the exact SMS-style summary from the original product
  brief — see `internal/format/sms_test.go`'s worked example.
- **`internal/vendor`** — the `Sender` interface every delivery channel implements, plus
  `LogSender`, the safe default that logs instead of actually sending. **No real SMS
  vendor is wired up yet** (see below) — this is intentional, not a placeholder bug.
- **`internal/consumer`** — routes a notification request to the right formatter, looks up
  the tenant's enabled channels, and fans out to each one's registered `Sender`.
- **`internal/store`** — `notification_channels` lookups.

## Status: no real SMS vendor configured

`cmd/notifier` wires `"sms"` to `vendor.LogSender` — every notification is logged, not
actually sent, until a real vendor (Twilio, AWS End User Messaging, etc.) is chosen and
implemented behind `vendor.Sender`. This is deliberate: shipping with a real, untested
vendor integration risked either blocking this repo on a vendor decision that wasn't part
of this task, or accidentally texting a real customer from a misconfigured environment.
See [CONTRIBUTING.md](CONTRIBUTING.md) for how to add one.

## Development

```bash
make test
make lint
make migrate-up   # against a local/test database only
```
