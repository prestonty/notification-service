package routing

import (
	"testing"

	"github.com/prestonty/notification-service/internal/event"
	"github.com/prestonty/notification-service/internal/notification"
)

func TestRouteChannels(t *testing.T) {
	r := New()

	tests := []struct {
		name     string
		event    event.Type
		expected []notification.Channel
	}{
		{
			name:     "PR merged routes to email",
			event:    event.PRMerged,
			expected: []notification.Channel{notification.ChannelEmail},
		},
		{
			name:     "deploy succeeded routes to email",
			event:    event.DeploySucceeded,
			expected: []notification.Channel{notification.ChannelEmail},
		},
		{
			name:     "deploy failed routes to email and SMS",
			event:    event.DeployFailed,
			expected: []notification.Channel{notification.ChannelEmail, notification.ChannelSMS},
		},
		{
			name:     "order ready routes to SMS",
			event:    event.OrderReady,
			expected: []notification.Channel{notification.ChannelSMS},
		},
		{
			name:     "unknown event defaults to email",
			event:    "UNKNOWN_EVENT",
			expected: []notification.Channel{notification.ChannelEmail},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channels := r.RouteChannels(tt.event)
			if len(channels) != len(tt.expected) {
				t.Fatalf("got %d channels, want %d", len(channels), len(tt.expected))
			}
			for i, ch := range channels {
				if ch != tt.expected[i] {
					t.Errorf("channel[%d] = %q, want %q", i, ch, tt.expected[i])
				}
			}
		})
	}
}
