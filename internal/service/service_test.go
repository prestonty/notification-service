package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/prestonty/notification-service/internal/event"
	"github.com/prestonty/notification-service/internal/notification"
	"github.com/prestonty/notification-service/internal/routing"
	"github.com/prestonty/notification-service/internal/template"
)

type mockSender struct {
	sent []notification.Notification
	err  error
}

func (m *mockSender) Send(ctx context.Context, n notification.Notification) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, n)
	return nil
}

func newTestService(email, sms *mockSender) *Service {
	providers := map[notification.Channel]notification.Sender{
		notification.ChannelEmail: email,
		notification.ChannelSMS:   sms,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return New(providers, routing.New(), template.New(), logger)
}

func TestProcessEvent(t *testing.T) {
	t.Run("PR_MERGED sends email", func(t *testing.T) {
		email := &mockSender{}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		results, err := svc.ProcessEvent(
			context.Background(),
			event.PRMerged,
			[]string{"user@example.com"},
			map[string]string{"pr_id": "42", "repo": "my-repo", "author": "alice"},
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Status != "sent" {
			t.Errorf("expected status sent, got %q", results[0].Status)
		}
		if results[0].Channel != "email" {
			t.Errorf("expected channel email, got %q", results[0].Channel)
		}
		if len(email.sent) != 1 {
			t.Fatalf("expected 1 email sent, got %d", len(email.sent))
		}
		if email.sent[0].Message != "PR #42 was merged in my-repo by alice." {
			t.Errorf("unexpected message: %s", email.sent[0].Message)
		}
	})

	t.Run("DEPLOY_FAILED sends email and SMS", func(t *testing.T) {
		email := &mockSender{}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		results, err := svc.ProcessEvent(
			context.Background(),
			event.DeployFailed,
			[]string{"oncall@example.com"},
			map[string]string{"service": "api", "environment": "prod", "log_url": "https://logs.example.com"},
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if len(email.sent) != 1 {
			t.Errorf("expected 1 email, got %d", len(email.sent))
		}
		if len(sms.sent) != 1 {
			t.Errorf("expected 1 sms, got %d", len(sms.sent))
		}
	})

	t.Run("multiple recipients", func(t *testing.T) {
		email := &mockSender{}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		results, err := svc.ProcessEvent(
			context.Background(),
			event.PRMerged,
			[]string{"a@test.com", "b@test.com", "c@test.com"},
			map[string]string{"pr_id": "1", "repo": "repo", "author": "bob"},
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		if len(email.sent) != 3 {
			t.Errorf("expected 3 emails, got %d", len(email.sent))
		}
	})

	t.Run("unknown event returns error", func(t *testing.T) {
		email := &mockSender{}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		_, err := svc.ProcessEvent(
			context.Background(),
			"FAKE_EVENT",
			[]string{"user@test.com"},
			map[string]string{},
			"",
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("missing required field returns error", func(t *testing.T) {
		email := &mockSender{}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		_, err := svc.ProcessEvent(
			context.Background(),
			event.PRMerged,
			[]string{"user@test.com"},
			map[string]string{"pr_id": "42"},
			"",
		)
		if err == nil {
			t.Fatal("expected error for missing required fields, got nil")
		}
	})

	t.Run("provider failure returns failed status", func(t *testing.T) {
		email := &mockSender{err: fmt.Errorf("connection refused")}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		results, err := svc.ProcessEvent(
			context.Background(),
			event.PRMerged,
			[]string{"user@test.com"},
			map[string]string{"pr_id": "1", "repo": "repo", "author": "bob"},
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results[0].Status != "failed" {
			t.Errorf("expected status failed, got %q", results[0].Status)
		}
		if results[0].Error != "connection refused" {
			t.Errorf("expected error message, got %q", results[0].Error)
		}
	})
}

func TestSendDirect(t *testing.T) {
	t.Run("sends to specified channel", func(t *testing.T) {
		email := &mockSender{}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		results, err := svc.SendDirect(
			context.Background(),
			notification.ChannelSMS,
			[]string{"5551234567"},
			"",
			"Your code is ready",
			"",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if len(sms.sent) != 1 {
			t.Errorf("expected 1 sms, got %d", len(sms.sent))
		}
		if len(email.sent) != 0 {
			t.Errorf("expected 0 emails, got %d", len(email.sent))
		}
	})

	t.Run("unknown channel returns error", func(t *testing.T) {
		email := &mockSender{}
		sms := &mockSender{}
		svc := newTestService(email, sms)

		_, err := svc.SendDirect(
			context.Background(),
			"pigeon",
			[]string{"user@test.com"},
			"",
			"Hello",
			"",
		)
		if err == nil {
			t.Fatal("expected error for unknown channel, got nil")
		}
	})
}
