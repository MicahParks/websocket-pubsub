package pubsub

// canceller is something that can be cancelled.
type canceller interface {
	Cancel()
}
