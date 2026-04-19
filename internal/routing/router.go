package routing

import (
	"github.com/prestonty/notification-service/internal/event"
	"github.com/prestonty/notification-service/internal/notification"
)

// Router determines which channels to use for a given event
type Router struct {
	rules map[event.Type][]notification.Channel
}

func New() *Router {
	return &Router {
		rules: map[event.Type][]notification.Channel {
			event.PRMerged:		{notification.ChannelEmail},
			event.DeploySucceeded: {notification.ChannelEmail},
			event.DeployFailed:    {notification.ChannelEmail, notification.ChannelSMS},
			event.OrderReady:      {notification.ChannelSMS},
		},
	}
}

// RouteChannels returns the channels for the given event type.
// Falls back to email if the event has no explicit rule.
func (r *Router) RouteChannels(eventType event.Type) []notification.Channel {
	channels, ok := r.rules[eventType]
	if !ok {
		return []notification.Channel{notification.ChannelEmail}
	}
	return channels
}
