package notification

import "context"

// Sender is implemented by every delivery provider (email, SMS, etc.).
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

