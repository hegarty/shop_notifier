# Contributing

See [shop_platform/CONTRIBUTING.md](https://github.com/hegarty/shop_platform/blob/main/CONTRIBUTING.md)
for the general workflow (PRs required, signed commits, required checks).

## Adding a real SMS vendor

Implement `vendor.Sender` (see `internal/vendor/vendor.go` and `internal/vendor/log.go`
for the interface and pattern) and register it in `cmd/notifier/main.go`'s `Senders` map.
No other code needs to change — `internal/consumer` only depends on the interface.

## Adding a new notification kind/formatter

`internal/consumer.Consumer.HandleRequest` switches on `Request.Kind`, currently only
`"analytics_result"`. A genuinely new kind (not just a new *job* under
`analytics_result` — see `handleAnalyticsResult`'s switch on `Job`) needs a new case there
and a new formatter in `internal/format`.

## Local development

```bash
make test
make lint
make migrate-up   # against a local/test database only
```
