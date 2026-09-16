// Command notifier consumes notifications.requested and delivers formatted
// messages to each tenant's configured channels. Ships with only a log
// Sender wired up (see internal/vendor/log.go) until a real SMS vendor is
// chosen — see README.md.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/hegarty/shop_platform/config"
	"github.com/hegarty/shop_platform/db"
	"github.com/hegarty/shop_platform/logging"
	"github.com/hegarty/shop_platform/otelx"
	"github.com/hegarty/shop_platform/redpanda"

	"github.com/hegarty/shop_notifier/internal/consumer"
	"github.com/hegarty/shop_notifier/internal/store"
	"github.com/hegarty/shop_notifier/internal/vendor"
)

// NotificationRequestsTopic mirrors shop_analytics's
// worker.NotificationRequestsTopic — see internal/consumer's package doc
// for why this isn't a shared import.
const NotificationRequestsTopic = "notifications.requested"

func main() {
	l := config.NewLoader()
	dbHost := l.String("DATABASE_HOST")
	dbPort := l.IntDefault("DATABASE_PORT", 5432)
	dbName := l.String("DATABASE_NAME")
	dbUser := l.String("DATABASE_USER")
	dbPassword := l.String("DATABASE_PASSWORD")
	redpandaBrokers := l.String("REDPANDA_BROKERS")
	otelEndpoint := l.StringDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	logLevel := l.StringDefault("LOG_LEVEL", "info")
	tenantID := l.String("TENANT_ID")
	tenantDisplayName := l.StringDefault("TENANT_DISPLAY_NAME", tenantID)
	if err := l.Err(); err != nil {
		slog.Error("configuration error", slog.Any("error", err))
		os.Exit(1)
	}

	logger := logging.New("shop-notifier", parseLevel(logLevel))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if otelEndpoint != "" {
		shutdown, err := otelx.Bootstrap(ctx, otelx.Config{
			ServiceName: "shop-notifier", ServiceVersion: "dev",
			Endpoint: otelEndpoint, Insecure: true,
		})
		if err != nil {
			logger.Error("otel bootstrap failed", slog.Any("error", err))
			os.Exit(1)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdown(shutdownCtx)
		}()
	}

	pool, err := db.Connect(ctx, db.Config{Host: dbHost, Port: dbPort, Database: dbName, User: dbUser}, dbPassword)
	if err != nil {
		logger.Error("db connect failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	c := &consumer.Consumer{
		Channels: store.NewChannelStore(pool),
		Senders: map[string]vendor.Sender{
			// Only "sms" has a configured channel type today, and even
			// that maps to the safe log Sender until a real vendor is
			// chosen. Add real Senders here (e.g. Twilio) behind the same
			// vendor.Sender interface — no other code needs to change.
			"sms": &vendor.LogSender{Logger: logger},
		},
		TenantDisplayNames: map[string]string{tenantID: tenantDisplayName},
	}

	kgoConsumer, err := redpanda.NewConsumer([]string{redpandaBrokers}, "shop-notifier", []string{NotificationRequestsTopic})
	if err != nil {
		logger.Error("redpanda consumer init failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer kgoConsumer.Close()

	logger.Info("shop-notifier started")
	err = kgoConsumer.Run(ctx, func(ctx context.Context, record *kgo.Record) error {
		var req consumer.Request
		if err := json.Unmarshal(record.Value, &req); err != nil {
			logger.Error("failed to decode notification request", slog.Any("error", err))
			return err
		}
		if err := c.HandleRequest(ctx, req); err != nil {
			logger.Error("notification delivery failed",
				slog.String("tenant_id", req.TenantID), slog.String("job", req.Job), slog.Any("error", err))
			return err
		}
		return nil
	})
	if err != nil && ctx.Err() == nil {
		logger.Error("notifier consumer stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
