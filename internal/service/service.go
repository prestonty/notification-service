package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prestonty/notification-service/internal/event"
	"github.com/prestonty/notification-service/internal/notification"
	"github.com/prestonty/notification-service/internal/routing"
	"github.com/prestonty/notification-service/internal/template"
)

// Result reports the outcome of a single notification dispatch
type Result struct {
	NotificationID	string `json:"notification_id"`
	Recipient		string `json:"recipient"`
	Channel			string `json:"channel"`
	Status			string `json:"status"`
	Error			string `json:"error,omitempty"`
}

// Service orchestrates the notification pipeline
type Service struct {
	providers	 map[notification.Channel]notification.Sender
	router		*routing.Router
	templates	*template.Engine
	logger		*slog.Logger
}

func New(
	providers 	map[notification.Channel]notification.Sender,
	router		*routing.Router,
	templates	*template.Engine,
	logger		*slog.Logger,
) *Service {
	return &Service {
		providers: providers,
		router: router,
		templates: templates,
		logger: logger,
	}
}

// ProcessEvent handles an event-based notification request
func (s *Service) ProcessEvent(
	ctx context.Context,
	eventType event.Type,
	recipients []string,
	data map[string]string,
	link string,
) ([]Result, error) {
	// 1. Look up template
	tmpl, err := event.Lookup(eventType)
	if err != nil {
		return nil, err
	}

	// 2. Validate required fields
	for _, key := range tmpl.Required {
		if _, ok := data[key]; !ok {
			return nil, fmt.Errorf("missing required field: %s", key)
		}
	}

	// 3. Render subject body
	subject, err := s.templates.Render(tmpl.Subject, data)
	if err != nil {
		return nil, fmt.Errorf("render subject: %w", err)
	}

	body, err := s.templates.Render(tmpl.Body, data)
	if err != nil {
		return nil, fmt.Errorf("render body: %w", err)
	}

	// 4. Determine channels
	channels := s.router.RouteChannels(eventType)

	// 5. Dispatch
	return s.dispatch(ctx, channels, recipients, subject, body, link), nil
}


// SendDirect handles a direct notification request
func (s *Service) SendDirect(
	ctx context.Context,
	channel notification.Channel,
	recipients []string,
	subject string,
	message string,
	link string,
) ([]Result, error) {
	if _, ok := s.providers[channel]; !ok {
		return nil, fmt.Errorf("no provider for channel: %s", channel)
	}

	return s.dispatch(ctx, []notification.Channel{channel}, recipients, subject, message, link), nil
}

func (s *Service) dispatch(
	ctx context.Context,
	channels []notification.Channel,
	recipients []string,
	subject string,
	message string,
	link string,
) []Result {
	var results []Result

	for _, recipient := range recipients {
		for _, ch := range channels {
			n := notification.Notification {
				ID:			fmt.Sprintf("%d", time.Now().UnixNano()),
				Channel: 	ch,
				Recipient: 	recipient,
				Subject: 	subject,
				Message: 	message,
				Link:		link,
				Status:		notification.StatusQueued,
				CreatedAt: time.Now(),
			}

			result := Result {
				NotificationID: n.ID,
				Recipient:		recipient,
				Channel:		string(ch),
			}

			provider, ok := s.providers[ch]
			if !ok {
				result.Status = "failed"
				result.Error = fmt.Sprintf("no provider for channel: %s", ch)
			} else if err := provider.Send(ctx, n); err != nil {
				s.logger.Error(
					"dispatch failed",
					"notification_id", n.ID,
					"channel", ch,
					"recipient", recipient,
					"error", err,
				)
				result.Status = "failed"
				result.Error = err.Error()
			} else {
				result.Status = "sent"
			}

			results = append(results, result)
		}
	}

	return results
}
