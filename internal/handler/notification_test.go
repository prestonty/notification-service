package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedKey    string
		expectedValue  string
	}{
		{
			name:           "valid notification",
			method:         http.MethodPost,
			body:           `{"type":"email","recipients":["user@example.com"],"subject":"Hello","message":"Test notification"}`,
			expectedStatus: http.StatusOK,
			expectedKey:    "status",
			expectedValue:  "accepted",
		},
		{
			name:           "missing type",
			method:         http.MethodPost,
			body:           `{"recipients":["user@example.com"],"message":"Hello"}`,
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
			expectedValue:  "type is required",
		},
		{
			name:           "missing message",
			method:         http.MethodPost,
			body:           `{"type":"email","recipients":["user@example.com"]}`,
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
			expectedValue:  "message is required",
		},
		{
			name:           "missing recipients",
			method:         http.MethodPost,
			body:           `{"type":"email","message":"Hello"}`,
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
			expectedValue:  "recipients is required",
		},
		{
			name:           "invalid JSON",
			method:         http.MethodPost,
			body:           `not json`,
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
			handler := NewNotificationHandler(svc)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/notifications", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/notifications", nil)
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
