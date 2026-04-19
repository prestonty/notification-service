package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/prestonty/notification-service/internal/notification"
	"github.com/prestonty/notification-service/internal/routing"
	"github.com/prestonty/notification-service/internal/service"
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

func newTestService() *service.Service {
	email := &mockSender{}
	sms := &mockSender{}
	providers := map[notification.Channel]notification.Sender{
		notification.ChannelEmail: email,
		notification.ChannelSMS:   sms,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return service.New(providers, routing.New(), template.New(), logger)
}

func TestEventHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedKey    string
		expectedValue  string
	}{
		{
			name:           "valid event",
			method:         http.MethodPost,
			body:           `{"event":"PR_MERGED","recipients":["user@example.com"],"data":{"pr_id":"42","repo":"my-repo","author":"alice"}}`,
			expectedStatus: http.StatusOK,
			expectedKey:    "status",
			expectedValue:  "accepted",
		},
		{
			name:           "missing event field",
			method:         http.MethodPost,
			body:           `{"recipients":["user@example.com"]}`,
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
			expectedValue:  "event is required",
		},
		{
			name:           "missing recipients",
			method:         http.MethodPost,
			body:           `{"event":"PR_MERGED","data":{"pr_id":"42","repo":"my-repo","author":"alice"}}`,
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
			expectedValue:  "recipients is required",
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           `{bad json`,
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
			expectedValue:  "invalid JSON",
		},
		{
			name:           "wrong HTTP method",
			method:         http.MethodGet,
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedKey:    "error",
			expectedValue:  "method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService()
			handler := NewEventHandler(svc)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/events", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/events", nil)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}

			var resp map[string]any
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			val, ok := resp[tt.expectedKey]
			if !ok {
				t.Fatalf("response missing key %q", tt.expectedKey)
			}
			if val != tt.expectedValue {
				t.Errorf("%s = %q, want %q", tt.expectedKey, val, tt.expectedValue)
			}
		})
	}
}
