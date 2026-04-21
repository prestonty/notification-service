package email

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prestonty/notification-service/internal/notification"
)

// Provider sends email notifications via the Mailgun HTTP API.
type Provider struct {
	domain string
	apiKey string
	from   string
	client *http.Client
	logger *slog.Logger
}

func New(domain, apiKey, from string, logger *slog.Logger) *Provider {
	return &Provider{
		domain: domain,
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

func (p *Provider) Send(ctx context.Context, n notification.Notification) error {
	endpoint := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", p.domain)

	form := url.Values{}
	form.Set("from", p.from)
	form.Set("to", n.Recipient)
	form.Set("subject", n.Subject)
	form.Set("text", n.Message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("mailgun: build request: %w", err)
	}
	req.SetBasicAuth("api", p.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("mailgun: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailgun: non-2xx status %d: %s", resp.StatusCode, string(body))
	}

	p.logger.Info(
		"mailgun: notification sent",
		"id", n.ID,
		"channel", n.Channel,
		"recipient", n.Recipient,
		"status", resp.StatusCode,
	)
	return nil
}
