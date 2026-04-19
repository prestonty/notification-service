package notification

import "time"

// Channel represents a delivery method
type Channel string

const (
	ChannelEmail	Channel = "email"
	ChannelSMS 		Channel = "sms"
)

// Status represents teh deliver status of a notification
type Status string

const (
	StatusQueued Status = "queued"
	StatusSent   Status = "sent"
	StatusFailed Status = "failed"
)

// Notification is the core domain entity
type Notification struct {
	ID        	string
	Channel   	Channel
	Recipient 	string
	Subject		string
	Message   	string
	Link		string
	Status    	Status
	CreatedAt 	time.Time
	UpdatedAt 	time.Time
}
