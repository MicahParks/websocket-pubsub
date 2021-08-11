package pubsub

import (
	"time"

	"github.com/gorilla/websocket"

	"github.com/MicahParks/pubsub/clients"
)

var (

	// closeDeadlineVar is the CloseDeadline as a variable instead of a constant.
	closeDeadlineVar = clients.CloseDeadline

	// defaultSubscriptionOptions contains the default options a subscription can have.
	defaultSubscriptionOptions = SubscriptionOptions{
		CloseDeadline:          &closeDeadlineVar,
		Events:                 nil,
		MessageBuffer:          10,
		PublisherBuffer:        1,
		SubscriberBuffer:       1,
		SubscriberWriteTimeout: &[]time.Duration{time.Second * 10}[0],
	}
)

// SubscriptionOptions represents information used to create a subscription that already has default values.
type SubscriptionOptions struct {

	// CloseDeadline is the time to wait to gracefully closing something. After that, it's closed regardless.
	CloseDeadline *time.Duration

	// Events is a channel of events for the subscription.
	Events chan<- Event

	// MessageBuffer is the internal channel buffer for messages for the subscription.
	MessageBuffer uint

	// PongDeadline is the time to wait for a pong message after a ping.
	PongDeadline *time.Duration

	// PublisherBuffer is the internal channel buffer for adding and removing publishers.
	PublisherBuffer uint

	// SubscriberBuffer is the internal channel buffer for adding and remove subscribers.
	SubscriberBuffer uint

	// SubscriberWriteTimeout is the time to wait for a message to be written to a subscriber. If the message is not
	// written in this amount of time, the subscriber is closed.
	SubscriberWriteTimeout *time.Duration
}

// Options represents all of the websocket pubsub information that already has default values.
type Options struct {

	// Options are the options for all subscriptions created by the service.
	Options *SubscriptionOptions

	// Upgrader is the websocket upgrader to when clients connect.
	Upgrader *websocket.Upgrader
}

// directClientOptions represents subscription information that already has default values.
type directClientOptions struct {
	closeDeadline *time.Duration
	events        chan<- Event
	pongDeadline  *time.Duration
}

// flattenOptions takes in a slice of Options, uses the highest index of their fields' values to create one Options.
func flattenOptions(options []Options) (option Options) {
	for _, opt := range options {
		if opt.Options != nil {
			option.Options = opt.Options
		}
		if opt.Upgrader != nil {
			option.Upgrader = opt.Upgrader
		}
	}
	if option.Options == nil {
		option.Options = &defaultSubscriptionOptions
	}
	if option.Upgrader == nil {
		option.Upgrader = &websocket.Upgrader{}
	}
	return option
}

// flattenSubscriptionOptions takes in a slice of SubscriptionOptions, uses the highest index of their fields' values to
// create one SubscriptionOptions.
func flattenSubscriptionOptions(options []SubscriptionOptions) (option SubscriptionOptions) {
	for _, opt := range options {
		if opt.CloseDeadline != nil {
			option.CloseDeadline = opt.CloseDeadline
		}
		if opt.Events != nil {
			option.Events = opt.Events
		}
		if opt.MessageBuffer != 0 {
			option.MessageBuffer = opt.MessageBuffer
		}
		if opt.PongDeadline != nil {
			option.PongDeadline = opt.PongDeadline
		}
		if opt.PublisherBuffer != 0 {
			option.PublisherBuffer = opt.PublisherBuffer
		}
		if opt.SubscriberBuffer != 0 {
			option.SubscriberBuffer = opt.SubscriberBuffer
		}
		if opt.SubscriberWriteTimeout != nil {
			option.SubscriberWriteTimeout = opt.SubscriberWriteTimeout
		}
	}
	if option.CloseDeadline == nil {
		option.CloseDeadline = defaultSubscriptionOptions.CloseDeadline
	}
	if option.Events == nil {
		option.Events = defaultSubscriptionOptions.Events
	}
	if option.MessageBuffer == 0 {
		option.MessageBuffer = defaultSubscriptionOptions.MessageBuffer
	}
	if option.PongDeadline == nil {
		deadline := clients.PongDeadline
		option.PongDeadline = &deadline
	}
	if option.PublisherBuffer == 0 {
		option.PublisherBuffer = defaultSubscriptionOptions.PublisherBuffer
	}
	if option.SubscriberBuffer == 0 {
		option.SubscriberBuffer = defaultSubscriptionOptions.SubscriberBuffer
	}
	if option.SubscriberWriteTimeout == nil {
		option.SubscriberWriteTimeout = defaultSubscriptionOptions.SubscriberWriteTimeout
	}
	return option
}

// flattenDirectClientOptions takes in a slice of directClientOptions, uses the highest index of their fields' values to
// create one directClientOptions.
func flattenDirectClientOptions(options []directClientOptions) (option directClientOptions) {
	for _, opt := range options {
		if opt.closeDeadline != nil {
			option.closeDeadline = opt.closeDeadline
		}
		if opt.events != nil {
			option.events = opt.events
		}
		if opt.pongDeadline != nil {
			option.pongDeadline = opt.pongDeadline
		}
	}
	if option.closeDeadline == nil {
		deadline := clients.CloseDeadline
		option.closeDeadline = &deadline
	}
	if option.pongDeadline == nil {
		deadline := clients.PongDeadline
		option.pongDeadline = &deadline
	}
	return option
}
