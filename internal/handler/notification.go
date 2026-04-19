package handler

import (
	"encoding/json"
	"net/http"

	"github.com/prestonty/notification-service/internal/notification"
	"github.com/prestonty/notification-service/internal/service"
)

type NotificationRequest struct {
	Type       string   `json:"type"`
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	Message    string   `json:"message"`
	Link       string   `json:"link"`
}

type NotificationHandler struct {
	svc *service.Service
}

func NewNotificationHandler(svc *service.Service) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if len(req.Recipients) == 0 {
		writeError(w, http.StatusBadRequest, "recipients is required")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	channel := notification.Channel(req.Type)

	results, err := h.svc.SendDirect(
		r.Context(),
		channel,
		req.Recipients,
		req.Subject,
		req.Message,
		req.Link,
	)

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "accepted",
		"results": results,
	})


}