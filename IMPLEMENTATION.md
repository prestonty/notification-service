# Implementation Guide

This document covers coding style, architecture patterns, provider extensibility, and event handling for the notification service.

---

## Table of Contents

- [Coding Style](#coding-style)
- [Architecture & Design Patterns](#architecture--design-patterns)
- [Project Structure](#project-structure)
- [How to Add a New Provider](#how-to-add-a-new-provider)
- [Event Handler System](#event-handler-system)
- [Direct Notification Flow](#direct-notification-flow)
- [Template Engine](#template-engine)
- [Channel Router](#channel-router)
- [Configuration](#configuration)
- [Error Handling & Logging](#error-handling--logging)
- [Testing Strategy](#testing-strategy)

---

## Coding Style

### General Rules

- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Use `gofmt` / `goimports` on every save. No exceptions.
- Exported names get doc comments. Unexported names get comments only when the logic isn't obvious.
- No global mutable state. Pass dependencies explicitly.
- Errors are values — handle them, don't ignore them. Never use `_` to discard an error unless you've documented why.

### Naming

```go
// Types: noun, PascalCase
type Notification struct { ... }
type EmailProvider struct { ... }

// Interfaces: verb or -er suffix, kept small
type Sender interface {
    Send(ctx context.Context, n Notification) error
}

// Constructors: New + type name, return the concrete type (not the interface)
func NewEmailProvider(cfg EmailConfig) *EmailProvider { ... }

// Methods: short receiver names (1-2 letters), consistent within a type
func (p *EmailProvider) Send(ctx context.Context, n Notification) error { ... }

// Package-level functions: verb + noun
func RouteChannels(event EventType) []Channel { ... }
func RenderTemplate(tmpl string, data map[string]string) (string, error) { ... }

// Constants: PascalCase for exported, camelCase or ALL_CAPS not used
const DefaultPort = 8080
```

### Package Design

- Packages should be small and focused around a single responsibility.
- Name packages as short, lowercase, singular nouns (`notification`, not `notifications` or `notificationService`).
- Avoid `util`, `common`, or `helpers` packages. Put functions next to the code that uses them.
- Internal packages under `/internal` are not importable by external code — use this to enforce boundaries.

### Context

- Every function that does I/O or could be cancelled takes `context.Context` as its first parameter.
- Never store `context.Context` in a struct field. Pass it through function calls.

```go
// Good
func (p *EmailProvider) Send(ctx context.Context, n Notification) error

// Bad
type EmailProvider struct {
    ctx context.Context // don't do this
}
```

### Error Handling

- Wrap errors with context using `fmt.Errorf("...: %w", err)` so callers can inspect the chain.
- Define sentinel errors for known failure modes that callers need to handle differently.
- Let unexpected errors bubble up — don't swallow them.

```go
var ErrTemplateNotFound = errors.New("template not found")

func LookupTemplate(event EventType) (Template, error) {
    tmpl, ok := templates[event]
    if !ok {
        return Template{}, fmt.Errorf("event %q: %w", event, ErrTemplateNotFound)
    }
    return tmpl, nil
}
```

---

## Architecture & Design Patterns

### Dependency Injection via Interfaces

This is the core architectural pattern. Every external dependency (email API, SMS API, template store, etc.) is represented by a small interface. Concrete implementations are injected at startup.

```go
// The service depends on abstractions, not concrete types
type Service struct {
    providers map[Channel]Sender
    router    Router
    templates TemplateEngine
    logger    *slog.Logger
}

func NewService(
    providers map[Channel]Sender,
    router Router,
    templates TemplateEngine,
    logger *slog.Logger,
) *Service {
    return &Service{
        providers: providers,
        router:    router,
        templates: templates,
        logger:    logger,
    }
}
```

Why this matters:
- **Testability**: Swap real providers for mocks/fakes in tests.
- **Extensibility**: Add new providers without modifying the service.
- **Decoupling**: The service doesn't know or care about SendGrid, Twilio, etc.

### Strategy Pattern for Providers

Each notification channel (email, SMS) is a strategy that implements the same `Sender` interface. The service selects which strategy to use at runtime based on the channel router's decision.

```go
// All providers implement this
type Sender interface {
    Send(ctx context.Context, n Notification) error
}

// The service dispatches to the right provider
func (s *Service) dispatch(ctx context.Context, channel Channel, n Notification) error {
    provider, ok := s.providers[channel]
    if !ok {
        return fmt.Errorf("no provider registered for channel %q", channel)
    }
    return provider.Send(ctx, n)
}
```

### Layered Architecture

Requests flow through well-defined layers. Each layer has a single responsibility and only calls the layer below it.

```
HTTP Handler (parse request, validate, return response)
       |
  Event Handler (map event -> template, resolve channels)
       |
    Service (orchestrate: render template, route, dispatch)
       |
   Provider (deliver notification via external API)
```

Rules:
- **Handlers** know about HTTP. Nothing else does.
- **Service** knows about business logic. It doesn't know about HTTP or provider specifics.
- **Providers** know about external APIs. They don't know about events or templates.

### Constructor Validation

Validate configuration and dependencies at construction time, not at call time. If a provider can't be created with valid config, fail fast at startup.

```go
func NewEmailProvider(cfg EmailConfig) (*EmailProvider, error) {
    if cfg.Host == "" {
        return nil, errors.New("email host is required")
    }
    if cfg.Port == 0 {
        return nil, errors.New("email port is required")
    }
    return &EmailProvider{cfg: cfg}, nil
}
```

---

## Project Structure

```
notification-service/
|-- cmd/
|   |-- notification-service/
|       |-- main.go              # Wiring: config, providers, server, graceful shutdown
|
|-- internal/
|   |-- config/
|   |   |-- config.go            # Load from env vars
|   |
|   |-- server/
|   |   |-- server.go            # HTTP server, middleware, route registration
|   |   |-- routes.go            # Route definitions
|   |
|   |-- handler/
|   |   |-- notification.go      # POST /notifications handler
|   |   |-- event.go             # POST /events handler
|   |   |-- health.go            # GET /health handler
|   |   |-- request.go           # Request DTOs + validation
|   |   |-- response.go          # Response DTOs
|   |
|   |-- event/
|   |   |-- registry.go          # Event type -> template mapping
|   |   |-- types.go             # EventType constants, EventRequest struct
|   |
|   |-- template/
|   |   |-- engine.go            # Template rendering logic
|   |
|   |-- routing/
|   |   |-- router.go            # Channel routing logic
|   |
|   |-- service/
|   |   |-- service.go           # Core orchestration: template -> route -> dispatch
|   |
|   |-- provider/
|   |   |-- sender.go            # Sender interface definition
|   |   |-- email/
|   |   |   |-- email.go         # Email provider (SMTP or API)
|   |   |-- sms/
|   |   |   |-- sms.go           # SMS provider
|   |   |-- noop/
|   |       |-- noop.go          # No-op provider for dev/testing
|   |
|   |-- notification/
|       |-- notification.go      # Notification model (the domain entity)
|
|-- tests/
|   |-- integration/             # Integration tests
|
|-- Plans.md
|-- IMPLEMENTATION.md
|-- go.mod
|-- go.sum
```

Key decisions:
- **One model package** (`internal/notification`) owns the `Notification` struct. No duplicates.
- **Handler package** is separate from the server — handlers contain request/response logic, the server wires routes and middleware.
- **Provider package** contains the `Sender` interface and each provider in its own sub-package. This keeps provider-specific dependencies isolated.

---

## How to Add a New Provider

Adding a new notification channel (e.g., Slack, push notifications) requires changes in exactly three places. No existing code needs modification (Open/Closed Principle).

### Step 1: Create the Provider

Create a new package under `internal/provider/`:

```
internal/provider/slack/
    slack.go
```

```go
package slack

import (
    "context"
    "fmt"
    "net/http"

    "github.com/prestonty/notification-service/internal/notification"
)

type Config struct {
    WebhookURL string
    Timeout    time.Duration
}

type Provider struct {
    cfg    Config
    client *http.Client
}

func New(cfg Config) (*Provider, error) {
    if cfg.WebhookURL == "" {
        return nil, errors.New("slack webhook URL is required")
    }
    timeout := cfg.Timeout
    if timeout == 0 {
        timeout = 10 * time.Second
    }
    return &Provider{
        cfg:    cfg,
        client: &http.Client{Timeout: timeout},
    }, nil
}

// Send implements the provider.Sender interface.
func (p *Provider) Send(ctx context.Context, n notification.Notification) error {
    // Build Slack payload, POST to webhook
    // ...
    return nil
}
```

### Step 2: Register the Channel Constant

In `internal/notification/notification.go` (or wherever channels are defined):

```go
const (
    ChannelEmail Channel = "email"
    ChannelSMS   Channel = "sms"
    ChannelSlack Channel = "slack"  // <- add this
)
```

### Step 3: Wire It Up in main.go

```go
func main() {
    // ... existing setup ...

    slackProvider, err := slack.New(slack.Config{
        WebhookURL: cfg.SlackWebhookURL,
    })
    if err != nil {
        log.Fatal("failed to create slack provider:", err)
    }

    providers := map[notification.Channel]provider.Sender{
        notification.ChannelEmail: emailProvider,
        notification.ChannelSMS:   smsProvider,
        notification.ChannelSlack: slackProvider,  // <- add this
    }

    svc := service.New(providers, router, tmplEngine, logger)
    // ...
}
```

### Step 4 (Optional): Update Channel Routing

If the new channel should be auto-selected for certain events, update the router:

```go
func (r *Router) RouteChannels(event event.Type) []notification.Channel {
    switch event {
    case event.DeployFailed:
        return []notification.Channel{notification.ChannelSlack, notification.ChannelEmail}
    default:
        return []notification.Channel{notification.ChannelEmail}
    }
}
```

That's it. The service, handlers, and template engine are untouched.

### Provider Checklist

When implementing a new provider, make sure it:

- [ ] Implements `Sender` interface
- [ ] Takes a config struct in its constructor
- [ ] Validates config at construction time (fail fast)
- [ ] Accepts `context.Context` and respects cancellation
- [ ] Uses an `*http.Client` with a timeout (never the default client)
- [ ] Wraps errors with context: `fmt.Errorf("slack: send: %w", err)`
- [ ] Has unit tests with a mock HTTP server (`httptest.NewServer`)

---

## Event Handler System

### Overview

The event handler is the entry point for event-driven notifications. Instead of the client specifying "send an email with this body," the client says "this thing happened" and the system figures out what to send and how.

```
Client sends:  { "event": "PR_MERGED", "data": {"pr_id": "123", "repo": "my-repo"} }
                    |
Event Handler:  looks up template for PR_MERGED
                    |
Template Engine: renders "PR #123 was merged in my-repo"
                    |
Channel Router:  decides -> [email]
                    |
Service:         dispatches to email provider
```

### Event Types & Registry

Events are defined as typed constants with an associated template:

```go
// internal/event/types.go

package event

type Type string

const (
    PRMerged         Type = "PR_MERGED"
    DeploySucceeded  Type = "DEPLOY_SUCCEEDED"
    DeployFailed     Type = "DEPLOY_FAILED"
    OrderReady       Type = "ORDER_READY"
)
```

```go
// internal/event/registry.go

package event

type Template struct {
    Subject  string            // for email subject lines
    Body     string            // template body with {placeholders}
    Required []string          // required data fields
}

var registry = map[Type]Template{
    PRMerged: {
        Subject:  "PR Merged: {repo}",
        Body:     "PR #{pr_id} was merged in {repo} by {author}.",
        Required: []string{"pr_id", "repo", "author"},
    },
    DeployFailed: {
        Subject:  "Deploy Failed: {service}",
        Body:     "Deploy of {service} to {environment} failed. See logs: {log_url}",
        Required: []string{"service", "environment", "log_url"},
    },
}

func Lookup(t Type) (Template, error) {
    tmpl, ok := registry[t]
    if !ok {
        return Template{}, fmt.Errorf("event %q: %w", t, ErrUnknownEvent)
    }
    return tmpl, nil
}
```

### Event Request Structure

```go
// internal/handler/request.go

type EventRequest struct {
    Event      string            `json:"event"      binding:"required"`
    Recipients []string          `json:"recipients"  binding:"required,min=1,dive,email"`
    Data       map[string]string `json:"data"`
    Link       string            `json:"link,omitempty"`
}
```

### Event Handler Flow

```go
// internal/handler/event.go

func (h *EventHandler) HandleEvent(c *gin.Context) {
    var req EventRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    // 1. Parse and validate the event type
    eventType := event.Type(req.Event)

    // 2. Delegate to the service layer (which handles template + routing + dispatch)
    results, err := h.service.ProcessEvent(c.Request.Context(), eventType, req.Recipients, req.Data, req.Link)
    if err != nil {
        // The service returns typed errors we can map to status codes
        status := mapErrorToStatus(err)
        c.JSON(status, errorResponse(err))
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "accepted",
        "results": results,
    })
}
```

### Service Layer (Orchestration)

The service ties it all together:

```go
// internal/service/service.go

func (s *Service) ProcessEvent(
    ctx context.Context,
    eventType event.Type,
    recipients []string,
    data map[string]string,
    link string,
) ([]Result, error) {
    // 1. Look up the template for this event
    tmpl, err := event.Lookup(eventType)
    if err != nil {
        return nil, err
    }

    // 2. Validate required data fields
    if err := validateData(tmpl.Required, data); err != nil {
        return nil, fmt.Errorf("missing required data: %w", err)
    }

    // 3. Render the message
    body, err := s.templates.Render(tmpl.Body, data)
    if err != nil {
        return nil, fmt.Errorf("render template: %w", err)
    }

    subject, err := s.templates.Render(tmpl.Subject, data)
    if err != nil {
        return nil, fmt.Errorf("render subject: %w", err)
    }

    // 4. Determine which channels to use
    channels := s.router.RouteChannels(eventType)

    // 5. Build and dispatch notifications
    var results []Result
    for _, recipient := range recipients {
        for _, channel := range channels {
            n := notification.Notification{
                ID:        generateID(),
                Channel:   channel,
                Recipient: recipient,
                Subject:   subject,
                Message:   body,
                Link:      link,
                Status:    notification.StatusQueued,
                CreatedAt: time.Now(),
            }

            err := s.dispatch(ctx, channel, n)
            result := Result{
                NotificationID: n.ID,
                Recipient:      recipient,
                Channel:        string(channel),
            }
            if err != nil {
                s.logger.Error("dispatch failed",
                    "notification_id", n.ID,
                    "channel", channel,
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

    return results, nil
}
```

### Adding a New Event

To support a new event type:

1. Add the constant in `internal/event/types.go`
2. Add the template mapping in `internal/event/registry.go`
3. Optionally update routing rules in `internal/routing/router.go`

No handler, service, or provider code changes needed.

---

## Direct Notification Flow

The `POST /notifications` endpoint bypasses the event system entirely. The client provides the fully formed message.

```go
// internal/handler/notification.go

type NotificationRequest struct {
    Type       string   `json:"type"       binding:"required,oneof=email sms"`
    Recipients []string `json:"recipients"  binding:"required,min=1"`
    Message    string   `json:"message"     binding:"required"`
    Subject    string   `json:"subject,omitempty"`
    Link       string   `json:"link,omitempty"`
}

func (h *NotificationHandler) HandleNotification(c *gin.Context) {
    var req NotificationRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, errorResponse(err))
        return
    }

    channel := notification.Channel(req.Type)

    results, err := h.service.SendDirect(
        c.Request.Context(),
        channel,
        req.Recipients,
        req.Subject,
        req.Message,
        req.Link,
    )
    if err != nil {
        c.JSON(mapErrorToStatus(err), errorResponse(err))
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "status":  "accepted",
        "results": results,
    })
}
```

---

## Template Engine

### Interface

```go
// internal/template/engine.go

type Engine interface {
    Render(template string, data map[string]string) (string, error)
}
```

### V1: Simple String Replacement

```go
type SimpleEngine struct{}

func (e *SimpleEngine) Render(tmpl string, data map[string]string) (string, error) {
    result := tmpl
    for key, val := range data {
        // Sanitize the value — strip any braces to prevent injection
        sanitized := strings.NewReplacer("{", "", "}", "").Replace(val)
        result = strings.ReplaceAll(result, "{"+key+"}", sanitized)
    }

    // Check for unreplaced placeholders
    if strings.Contains(result, "{") && strings.Contains(result, "}") {
        return "", fmt.Errorf("template has unreplaced placeholders: %s", result)
    }

    return result, nil
}
```

### Future: Go text/template

When simple replacement isn't enough, swap `SimpleEngine` for a `GoTemplateEngine` that uses `text/template`. The `Engine` interface stays the same — no other code changes.

---

## Channel Router

### Interface

```go
// internal/routing/router.go

type Router interface {
    RouteChannels(eventType event.Type) []notification.Channel
}
```

### V1: Static Rules

```go
type StaticRouter struct {
    rules map[event.Type][]notification.Channel
}

func NewStaticRouter() *StaticRouter {
    return &StaticRouter{
        rules: map[event.Type][]notification.Channel{
            event.PRMerged:        {notification.ChannelEmail},
            event.DeployFailed:    {notification.ChannelEmail, notification.ChannelSMS},
            event.DeploySucceeded: {notification.ChannelEmail},
            event.OrderReady:      {notification.ChannelSMS},
        },
    }
}

func (r *StaticRouter) RouteChannels(eventType event.Type) []notification.Channel {
    channels, ok := r.rules[eventType]
    if !ok {
        return []notification.Channel{notification.ChannelEmail} // default fallback
    }
    return channels
}
```

---

## Configuration

Use environment variables loaded into a typed config struct at startup.

```go
// internal/config/config.go

type Config struct {
    Port int

    EmailHost string
    EmailPort int
    EmailUser string
    EmailPass string

    SMSApiKey string
    SMSApiURL string
}

func Load() (*Config, error) {
    port, _ := strconv.Atoi(getEnv("PORT", "8080"))

    cfg := &Config{
        Port:      port,
        EmailHost: os.Getenv("EMAIL_HOST"),
        EmailPort: parseIntOr(os.Getenv("EMAIL_PORT"), 587),
        EmailUser: os.Getenv("EMAIL_USER"),
        EmailPass: os.Getenv("EMAIL_PASS"),
        SMSApiKey: os.Getenv("SMS_API_KEY"),
        SMSApiURL: os.Getenv("SMS_API_URL"),
    }
    return cfg, nil
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

---

## Error Handling & Logging

### Structured Logging

Use Go's `log/slog` (standard library, Go 1.21+). No external logging library needed.

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

logger.Info("notification dispatched",
    "notification_id", n.ID,
    "channel", channel,
    "recipient", n.Recipient,
)

logger.Error("provider failed",
    "notification_id", n.ID,
    "channel", channel,
    "error", err,
)
```

### HTTP Error Mapping

Map domain errors to HTTP status codes in the handler layer only:

```go
func mapErrorToStatus(err error) int {
    switch {
    case errors.Is(err, event.ErrUnknownEvent):
        return http.StatusBadRequest
    case errors.Is(err, template.ErrRenderFailed):
        return http.StatusInternalServerError
    case errors.Is(err, ErrNoProvider):
        return http.StatusInternalServerError
    default:
        return http.StatusInternalServerError
    }
}
```

### Graceful Shutdown

```go
// in main.go

srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: router}

go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal("server error:", err)
    }
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    log.Fatal("forced shutdown:", err)
}
```

---

## Testing Strategy

### Unit Tests

Each package has its own `_test.go` files. Use interfaces to mock dependencies.

```go
// Mock sender for testing the service layer
type mockSender struct {
    sent []notification.Notification
    err  error
}

func (m *mockSender) Send(ctx context.Context, n notification.Notification) error {
    m.sent = append(m.sent, n)
    return m.err
}
```

### What to Test

| Layer          | What to test                                       |
| -------------- | -------------------------------------------------- |
| Handler        | Request validation, status codes, response shape    |
| Event Registry | Known events resolve, unknown events return error   |
| Template       | Placeholder replacement, missing data, injection    |
| Router         | Correct channels for each event, default fallback   |
| Service        | Full orchestration with mocked providers            |
| Provider       | HTTP request shape, error handling (use httptest)    |

### Integration Tests

Test the full HTTP flow with real HTTP requests against a running server, but with mocked providers:

```go
func TestEventEndToEnd(t *testing.T) {
    mock := &mockSender{}
    svc := service.New(
        map[notification.Channel]provider.Sender{
            notification.ChannelEmail: mock,
        },
        routing.NewStaticRouter(),
        template.NewSimpleEngine(),
        slog.Default(),
    )
    handler := handler.NewEventHandler(svc)
    router := setupTestRouter(handler)

    body := `{"event":"PR_MERGED","recipients":["user@test.com"],"data":{"pr_id":"42","repo":"my-repo","author":"alice"}}`
    req := httptest.NewRequest("POST", "/events", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    assert.Equal(t, 200, w.Code)
    assert.Len(t, mock.sent, 1)
    assert.Contains(t, mock.sent[0].Message, "PR #42 was merged in my-repo")
}
```
