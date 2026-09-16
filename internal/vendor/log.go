package vendor

import (
	"context"
	"log/slog"
)

// LogSender is the safe default Sender: it logs what would have been sent
// instead of calling a real provider. Used until an SMS vendor is chosen
// (see README.md) — deploying with this Sender never risks accidentally
// texting a real customer from a misconfigured environment.
type LogSender struct {
	Logger *slog.Logger
}

func (s *LogSender) Send(ctx context.Context, destination, body string) error {
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "notification (log vendor — not actually delivered)",
		slog.String("destination", destination),
		slog.String("body", body),
	)
	return nil
}
