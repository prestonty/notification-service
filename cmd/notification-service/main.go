package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	config "github.com/prestonty/notification-service/internal"
	"github.com/prestonty/notification-service/internal/handler"
	"github.com/prestonty/notification-service/internal/notification"
	"github.com/prestonty/notification-service/internal/provider/noop"
	"github.com/prestonty/notification-service/internal/routing"
	"github.com/prestonty/notification-service/internal/service"
	"github.com/prestonty/notification-service/internal/template"
)

func main() {
    cfg := config.Load()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Providers (Use noop for now)
    noopProvider := noop.New(logger)
    providers := map[notification.Channel]notification.Sender{
        notification.ChannelEmail:  noopProvider,
        notification.ChannelSMS:    noopProvider,
    }

    // Build service
	router := routing.New()
	tmplEngine := template.New()
	svc := service.New(providers, router, tmplEngine, logger)

    // Register routes
	eventHandler := handler.NewEventHandler(svc)
	notifHandler := handler.NewNotificationHandler(svc)

    mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
    mux.Handle("/events", eventHandler)
    mux.Handle("/notifications", notifHandler)

    addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("server starting", "addr", addr)
	log.Fatal(http.ListenAndServe(addr, mux))

}
