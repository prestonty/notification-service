package noop

import (
	"context"
	"log/slog"

	"github.com/prestonty/notification-service/internal/notification"
)

// Provider discards notification. User for local dev and testing
type Provider stuct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Provider {
	return &Provider{logger: logger}
}

func (p *Provider) Send(ctx context.Context, n notification.Notification) error {
	p.logger.Info("noop: notification sent",
		"id", n.ID,
		"channel", n.Channel,
		"recipient", n.Recipient,
		"message", n.Message,
	)
	return nil
}