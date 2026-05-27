package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/prestonty/notification-service/internal/notification"
)

// Provider sends email notifications via the Resend HTTP API.
type Provider struct {
	apiKey string
	from   string
	client *http.Client
	logger *slog.Logger
}

type sendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text"`
}

func New(apiKey, from string, logger *slog.Logger) *Provider {
	return &Provider{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

func (p *Provider) Send(ctx context.Context, n notification.Notification) error {
	body, err := json.Marshal(sendRequest{
		From:    p.from,
		To:      []string{n.Recipient},
		Subject: n.Subject,
		Text:    n.Message,
	})
	if err != nil {
		return fmt.Errorf("resend: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend: non-2xx status %d: %s", resp.StatusCode, string(respBody))
	}

	p.logger.Info(
		"resend: notification sent",
		"id", n.ID,
		"channel", n.Channel,
		"recipient", n.Recipient,
		"status", resp.StatusCode,
	)
	return nil
}
