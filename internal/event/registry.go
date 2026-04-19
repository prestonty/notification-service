package event

import (
	"errors"
	"fmt"
)

var ErrUnknownEvent = errors.New("unknown event")

// Template holds the message templates for an event
type Template struct {
	Subject		string
	Body		string
	Required	[]string
}

var registry = map[Type]Template {
	PRMerged: {
		Subject: 	"PR Merged: {repo}",
		Body:		"PR #{pr_id} was merged in {repo} by {author}.",
		Required:	[]string{"pr_id", "repo", "author"},
	},
	DeploySucceeded: {
		Subject:  "Deploy Succeeded: {service}",
		Body:     "Deploy of {service} to {environment} succeeded.",
		Required: []string{"service", "environment"},
	},
	DeployFailed: {
		Subject:  "Deploy Failed: {service}",
		Body:     "Deploy of {service} to {environment} failed. See logs: {log_url}",
		Required: []string{"service", "environment", "log_url"},
	},
	OrderReady: {
		Subject:  "Your order is ready",
		Body:     "Your order #{order_id} is ready for pickup at {location}.",
		Required: []string{"order_id", "location"},
	},
}

// Lookup returns the template for the given event type.
func Lookup(t Type) (Template, error) {
	tmpl, ok := registry[t]
	if !ok {
		return Template{}, fmt.Errorf("event %q: %w", t, ErrUnknownEvent)
	}
	return tmpl, nil
}
