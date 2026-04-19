# Notification Service (Go) — Plan

## Overview

This microservice is responsible for sending notifications to users via multiple channels.

For the initial version, we will support:

* Email notifications
* SMS (text message) notifications

The service is designed to be simple, extensible, and easily integrated into other applications.

---

## Goals (V1 Scope)

### Core Features

* Send read-only notifications
* Send notifications with links
* Send email notifications
* Send SMS notifications
* Accept event-based triggers
* Template-based message generation
* Channel routing (decide how users are notified)

### Non-Goals (for now)

* Push/browser notifications
* Scheduled/timed notifications
* Interactive state-changing notifications
* Full event bus integration (Kafka, etc.)

---

## High-Level Architecture

```
Client App → API Layer → Event Handler → Template Engine → Channel Router → Notification Service → Providers
```

---

## Components

### 1. API Layer

* Accepts requests (`/notifications`, `/events`)
* Validates input
* Routes to handlers

---

### 2. Event Handler Layer

* Receives domain events (e.g. `PR_MERGED`)
* Maps events → notification templates
* Passes data to template engine

---

### 3. Template Engine

#### Purpose

Generate dynamic messages using event data.

#### Example Template

```
"PR #{pr_id} was merged in {repo}"
```

#### Example Input

```json
{
  "pr_id": 123,
  "repo": "notification-service"
}
```

#### Output

```
"PR #123 was merged in notification-service"
```

#### Implementation Notes

* Start simple (string replacement)
* Later upgrade to:

  * Go `text/template`
  * or more advanced templating

---

### 4. Channel Router

#### Purpose

Decide **how** to notify the user.

#### Example Inputs

* Event type
* User preferences (future)
* System defaults

#### Example Logic

```go
if event == PR_MERGED {
    return []string{"email"}
}
```

#### Future Enhancements

* User notification preferences
* Fallback logic (email fails → SMS)
* Priority-based routing

---

### 5. Notification Service (Core Logic)

* Receives fully built notification
* Dispatches to providers

---

### 6. Providers

#### Email Provider

* SMTP or external API
* Sends formatted email

#### SMS Provider

* Sends short messages
* Handles length constraints

---

## API Design

### 1. Direct Notification

POST /notifications

```json
{
  "type": "email | sms",
  "recipients": ["user@example.com"],
  "message": "Your item is ready!",
  "link": "https://example.com/order/123"
}
```

---

### 2. Event-Based Trigger (Recommended)

POST /events

```json
{
  "event": "PR_MERGED",
  "recipients": ["user@example.com"],
  "data": {
    "pr_id": 123,
    "repo": "my-repo"
  }
}
```

---

## Event Handling System

### Event Flow

```
Your App → /events → Event Handler → Template Engine → Channel Router → Notification Service → Provider → User
```

---

## Event → Template Mapping

```go
type EventType string

const (
    PRMerged EventType = "PR_MERGED"
)

type NotificationTemplate struct {
    Template string
}
```

```go
var EventTemplates = map[EventType]NotificationTemplate{
    PRMerged: {
        Template: "PR #{pr_id} was merged in {repo}",
    },
}
```

---

## Channel Routing Strategy

```go
type Channel string

const (
    Email Channel = "email"
    SMS   Channel = "sms"
)
```

```go
func RouteChannels(event EventType) []Channel {
    switch event {
    case PRMerged:
        return []Channel{Email}
    default:
        return []Channel{Email}
    }
}
```

---

## Project Structure

```
/cmd
  /server

/internal
  /api
  /event
  /template      # template engine
  /routing       # channel routing logic
  /service
  /providers
    /email
    /sms
  /models
  /config

/tests
```

---

## Data Models

```go
type Notification struct {
    ID         string
    Type       string
    Recipients []string
    Message    string
    Link       string
    CreatedAt  time.Time
}
```

---

## Service Flow (Event-Based)

1. Receive `/events` request
2. Validate event
3. Lookup template
4. Render message (template engine)
5. Determine channels (router)
6. Send via providers
7. Return response

---

## Error Handling

* Invalid input → 400
* Unknown event → 400
* Template failure → 500
* Provider failure → 500
* Log all failures

---

## Testing Plan

### Unit Tests

* Event mapping
* Template engine
* Channel routing
* Service logic
* Provider interfaces (mocked)

### Integration Tests

* `/events` endpoint
* End-to-end notification flow

---

## Configuration

```
PORT=8080

EMAIL_HOST=
EMAIL_PORT=
EMAIL_USER=
EMAIL_PASS=

SMS_API_KEY=
SMS_API_URL=
```

---

## Future Enhancements

* Event bus integration
* Push notifications
* Scheduled notifications
* Retry queues
* Database-driven templates
* User preferences
* Rate limiting
* Observability

---

## Example Use Cases

* PR merged notification
* Order ready for pickup
* Purchase confirmation
* Reminder notifications

---

## Notes

* Keep providers simple (just send)
* Template engine should be isolated
* Routing should not depend on providers
* Avoid hardcoding logic across layers
* Start simple, evolve incrementally

