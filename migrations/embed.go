// Package migrations embeds shop_notifier's schema migration files.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
