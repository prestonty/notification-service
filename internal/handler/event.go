package handler

import (
	"encoding/json"
	"net/http"

	"github.com/prestonty/notification-service/internal/event"
	"github.com/prestonty/notification-service/internal/service"
)

type EventRequest struct {
	Event		string				`json:"event"`
	Recipients 	[]string			`json:"recipients"`
	Data       map[string]string 	`json:"data"`
	Link       string            	`json:"link"`
}

type EventHandler struct {
	svc *service.Service
}

func NewEventHandler(svc *service.Service) *EventHandler {
	return &EventHandler{svc: svc}
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Event == "" {
		writeError(w, http.StatusBadRequest, "event is required")
		return
	}
	if len(req.Recipients) == 0 {
		writeError(w, http.StatusBadRequest, "recipients is required")
		return
	}

	// Calls the ProcessEvent function in internal/service/service.go
	results, err := h.svc.ProcessEvent(
		r.Context(),
		event.Type(req.Event),
		req.Recipients,
		req.Data,
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