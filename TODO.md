# TODO: Middleware, Mailgun Email Provider, Graceful Shutdown

## Context

The notification service has a working end-to-end pipeline with full test coverage. The server lacks graceful shutdown, there's no request logging or tracing, and all notifications go to a noop provider. This plan adds middleware, a Mailgun email provider, and graceful shutdown.

## Implementation Order

1. **Middleware** — additive, no dependencies on other new code
2. **Mailgun email provider** — replaces noop for email channel
3. **Graceful shutdown** — touches main.go last, after middleware and provider are wired

---

## 1. Middleware

**Create:** `internal/middleware/requestid.go`, `internal/middleware/logging.go`

Both use the standard `func(http.Handler) http.Handler` pattern.

**requestid.go:**
- Generate 32-char hex ID via `crypto/rand`
- Honor incoming `X-Request-ID` header if present
- Set ID on response header and in request context
- Export `GetRequestID(ctx)` helper

**logging.go:**
- Closure over `*slog.Logger`
- Wrap `http.ResponseWriter` with a `statusWriter` to capture status code
- Log: method, path, status, duration_ms, request_id

**Wire in main.go:**
```go
var h http.Handler = mux
h = middleware.Logging(logger)(h)
h = middleware.RequestID(h)
```
Request flow: RequestID -> Logging -> mux -> handler

---

## 2. Mailgun Email Provider

**Create:** `internal/provider/email/email.go`
**Modify:** `internal/config.go`, `cmd/notification-service/main.go`

Mailgun uses a simple HTTP API — no SDK needed, just `net/http`.

**How Mailgun's API works:**
- POST to `https://api.mailgun.net/v3/{domain}/messages`
- Basic Auth with username `api` and password = API key
- Form-encoded body: `from`, `to`, `subject`, `text`

**Config changes to `internal/config.go`:**
```go
MailgunDomain string  // env: MAILGUN_DOMAIN
MailgunAPIKey string  // env: MAILGUN_API_KEY
MailgunFrom   string  // env: MAILGUN_FROM (e.g. "notifications@yourdomain.com")
```
Remove the old `EmailHost`, `EmailPort`, `EmailUser`, `EmailPass` fields (no longer needed).

**email.go:**
- Constructor: `New(domain, apiKey, from, logger)`
- `Send()` builds HTTP POST to Mailgun API with `http.NewRequestWithContext`
- Basic Auth: username `api`, password = API key
- Form-encoded: from, to, subject, text
- `*http.Client` with 30-second timeout
- Error handling: read response body on non-2xx, wrap error

**Conditional wiring in main.go:**
- `MailgunDomain` + `MailgunAPIKey` set -> real email provider
- Otherwise -> noop provider
- Log which provider is active at startup

SMS stays on noop for now.

---

## 3. Graceful Shutdown

**Modify:** `cmd/notification-service/main.go`

- Create `*http.Server` with middleware-wrapped handler
- Start `ListenAndServe` in a goroutine
- Listen for `SIGINT`/`SIGTERM` via `os/signal`
- Call `srv.Shutdown(ctx)` with 10-second timeout
- Log shutdown events

---

## Files Summary

| Action | File |
|--------|------|
| Create | `internal/middleware/requestid.go` |
| Create | `internal/middleware/logging.go` |
| Create | `internal/provider/email/email.go` |
| Modify | `internal/config.go` |
| Modify | `cmd/notification-service/main.go` |

## Verification

1. `go build ./...` — compiles clean
2. `go test ./...` — all tests pass
3. Start server, send request, confirm:
   - JSON log line with request_id, method, path, status, duration
   - `X-Request-ID` in response header
   - Noop provider still works without env vars
4. Ctrl+C the server, confirm graceful shutdown logs
