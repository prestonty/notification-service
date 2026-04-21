package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	config "github.com/prestonty/notification-service/internal"
	"github.com/prestonty/notification-service/internal/handler"
	"github.com/prestonty/notification-service/internal/middleware"
	"github.com/prestonty/notification-service/internal/notification"
	"github.com/prestonty/notification-service/internal/provider/email"
	"github.com/prestonty/notification-service/internal/provider/noop"
	"github.com/prestonty/notification-service/internal/routing"
	"github.com/prestonty/notification-service/internal/service"
	"github.com/prestonty/notification-service/internal/template"
)

func main() {
	if err := godotenv.Load(); err != nil {
    	logger.Info(".env not found, using shell env vars")
	}
	
    cfg := config.Load()
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Providers
    noopProvider := noop.New(logger)

    var emailProvider notification.Sender = noopProvider
    if cfg.MailgunDomain != "" && cfg.MailgunAPIKey != "" {
        emailProvider = email.New(cfg.MailgunDomain, cfg.MailgunAPIKey, cfg.MailgunFrom, logger)
        logger.Info("email provider: mailgun", "domain", cfg.MailgunDomain)
    } else {
        logger.Info("email provider: noop (MAILGUN_DOMAIN/MAILGUN_API_KEY unset)")
    }

    providers := map[notification.Channel]notification.Sender{
        notification.ChannelEmail: emailProvider,
        notification.ChannelSMS:   noopProvider,
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

	var h http.Handler = mux
	h = middleware.Logging(logger)(h)
	h = middleware.RequestID(h)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.Port),
		Handler: h,
	}

	// Start server in a goroutine so main can wait on signals
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Block until either the server dies or we get SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	select {
		case err := <- serverErr:
			logger.Error("server failed", "err", err)
			os.Exit(1)
		case sig := <- quit:
			logger.Info("shutdown signal received", "signal", sig.String())
	}

	// 10s window for in-flight requests to finish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")

}
