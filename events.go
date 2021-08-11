package pubsub

const (

	// EventTypePublisherLeft is an event that indicates a publisher has left the subscription.
	EventTypePublisherLeft EventType = 1

	// EventTypePublisherJoined is an event that indicates a publisher has joined the subscription.
	EventTypePublisherJoined EventType = 2

	// EventTypeSubscriberLeft is an event that indicates a subscriber has left the subscription.
	EventTypeSubscriberLeft EventType = 3

	// EventTypeSubscriberJoined is an event that indicates a subscriber has joined the subscription.
	EventTypeSubscriberJoined EventType = 4

	// EventTypeWebsocketUpgradeFailed is an event that indicates a websocket upgrade failed.
	EventTypeWebsocketUpgradeFailed EventType = 5

	// EventTypeBadHTTPHeaders is an event that indicates a incoming request didn't have appropriate HTTP headers.
	EventTypeBadHTTPHeaders EventType = 6
)

// EventType is an enum that indicates what kind of event has happened.
type EventType uint

// Event represents an event for a subscription.
type Event struct {

	// ClientID is the unique client ID for a client to the subscription that the event is for.
	ClientID string `json:"clientID"`

	// SubscriptionTopic is the topic of the subscription that generated this event. (The URL path.)
	SubscriptionTopic string `json:"subscriptionTopic"`

	// Type is an enum that indicates what kind of event has happened.
	Type EventType `json:"type"`
}
