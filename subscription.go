package pubsub

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// subscription represents a subscription that may have publishers and subscribers. Publishers send messages to all
// subscribers.
type subscription struct {
	cancel                 context.CancelFunc
	closeDeadline          time.Duration
	closeOnce              sync.Once
	ctx                    context.Context
	deletePublisher        chan *publisher
	deleteSubscriber       chan *subscriber
	done                   chan struct{}
	events                 chan<- Event
	newPublisher           chan *publisher
	newSubscriber          chan *subscriber
	publish                chan *websocket.PreparedMessage
	publishers             map[*publisher]struct{}
	subscriberWriteTimeout time.Duration
	subscribers            map[*subscriber]struct{}
}

// newSubscription creates a new subscription and launches the appropriate goroutines.
func newSubscription(ctx context.Context, options ...SubscriptionOptions) (subscript *subscription) {

	// Wrap the context with a cancellation.
	wrappedCtx, cancel := context.WithCancel(ctx)

	// Flatten the options.
	option := flattenSubscriptionOptions(options)

	// Create the subscription.
	subscript = &subscription{
		cancel:                 cancel,
		closeDeadline:          *option.CloseDeadline,
		ctx:                    wrappedCtx,
		deleteSubscriber:       make(chan *subscriber, option.SubscriberBuffer),
		deletePublisher:        make(chan *publisher, option.PublisherBuffer),
		done:                   make(chan struct{}),
		events:                 option.Events,
		subscribers:            make(map[*subscriber]struct{}),
		newSubscriber:          make(chan *subscriber, option.SubscriberBuffer),
		newPublisher:           make(chan *publisher, option.PublisherBuffer),
		publishers:             make(map[*publisher]struct{}),
		publish:                make(chan *websocket.PreparedMessage, option.MessageBuffer),
		subscriberWriteTimeout: *option.SubscriberWriteTimeout,
	}

	// Launch a separate goroutine that will handle publications, subscribers, and death.
	go subscript.run()

	return subscript
}

// Close closes the subscription and all connections to subscribers.
func (s *subscription) Close() {

	// Only close the subscription once.
	s.closeOnce.Do(func() {

		// Tell the separate goroutine to end.
		s.cancel()

		// Wait for the writer goroutine to exit.
		select {
		case <-s.done:
		case <-time.After(s.closeDeadline):
		}

		// Close the connection to all clients.
		for pub := range s.publishers {
			pub.Cancel()
		}
		for sub := range s.subscribers {
			sub.Cancel()
		}
	})
}

// publishMessage sends a message to all subscribers.
func (s *subscription) publishMessage(prepared *websocket.PreparedMessage) {

	// Iterate through the subscribers and send the published message.
	for sub := range s.subscribers {
		select {
		case sub.send <- prepared:
		default:
			sub := sub
			go func() {
				select {

				// Send the publication, if the subscriber will take it.
				case sub.send <- prepared:

				// The subscriber failed to take the publication.
				case <-time.After(s.subscriberWriteTimeout):
					sub.Cancel()
				}
			}()
		}
	}
}

// run is the goroutine launched by calling newSubscription. It handles messages, subscriptions, and death.
//
// All map operations are handled in this goroutine so there are no async problems.
func (s *subscription) run() {

	// Use the done channel to confirm the close messages will not be sent after/during close time.
	defer func() {
		close(s.done)
	}()

	// Handle messages and subscriptions until death.
	for {
		select {

		// When dead, clean up this goroutine.
		case <-s.ctx.Done():
			return

		// When a new subscriber comes in, keep track of them.
		case sub := <-s.newSubscriber:
			s.subscribers[sub] = struct{}{}

		// When a subscriber leaves, remove them.
		case sub := <-s.deleteSubscriber:
			delete(s.subscribers, sub)

		// When a message is published, send it to the subscribers.
		case prepared := <-s.publish:
			s.publishMessage(prepared)

		// When a new publisher comes in, keep track of them.
		case pub := <-s.newPublisher:
			s.publishers[pub] = struct{}{}

		// When a publisher leaves, remove them.
		case pub := <-s.deletePublisher:
			delete(s.publishers, pub)
		}
	}
}
