package pubsub

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"

	"github.com/MicahParks/pubsub/clients"
)

// publisher holds all the required information for a publisher to the pubsub system.
type publisher struct {
	cancel       context.CancelFunc
	conn         *websocket.Conn
	ctx          context.Context
	events       chan<- Event
	done         chan struct{}
	name         string
	pongDeadline time.Duration
	publish      chan<- *websocket.PreparedMessage
	topic        string
}

// newPublisher creates a new publisher and launches the appropriate goroutines.
func newPublisher(ctx context.Context, name string, conn *websocket.Conn, publish chan<- *websocket.PreparedMessage, topic string, options ...directClientOptions) (pub *publisher) {

	// Wrap the context in a cancel function.
	pubCtx, cancel := context.WithCancel(ctx)

	// Flatten the options.
	option := flattenDirectClientOptions(options)

	// Create the publisher.
	pub = &publisher{
		cancel:       cancel,
		conn:         conn,
		ctx:          pubCtx,
		done:         make(chan struct{}),
		events:       option.events,
		name:         name + "-" + uuid.NewV4().String(),
		pongDeadline: *option.pongDeadline,
		publish:      publish,
		topic:        topic,
	}

	// Close the client when the context expires.
	go func() {

		// Report the event when this function returns.
		if pub.events != nil {
			event := Event{
				ClientID:          pub.name,
				SubscriptionTopic: pub.topic,
				Type:              EventTypePublisherLeft,
			}
			defer func() {
				select {
				case pub.events <- event:
				case <-time.After(*option.closeDeadline):
				}
			}()
		}

		// Wait for the client's context to expire.
		<-pubCtx.Done()

		// Wait for the writer goroutines to exit.
		select {
		case <-pub.done:
		case <-time.After(*option.closeDeadline):
		}

		// Close the publisher's websocket.
		_ = clients.CloseWebSocket(pub.conn, pub.pongDeadline) // Ignore any error.
	}()

	// Launch the publisher's pumping goroutines.
	go pub.readPump()
	go pub.writePump()

	return pub
}

// Cancel implements the canceller interface.
func (p *publisher) Cancel() {
	p.cancel()
}

// readPump is the single goroutine for reading websocket messages from the publisher. It handles pings, pongs, and
// published messages.
func (p *publisher) readPump() {

	// If this goroutine is the first to exit, close the publisher.
	defer p.Cancel()

	// Read the published messages and publish them to the subscription.
	var err error
	var messageType int
	for {

		// Read the next message from the publisher.
		var message []byte
		if messageType, message, err = p.conn.ReadMessage(); err != nil {

			// End the goroutine.
			break
		}

		// Turn the received message into a prepared message for easier & faster transport.
		var prepared *websocket.PreparedMessage
		if prepared, err = websocket.NewPreparedMessage(messageType, message); err != nil {

			// End the goroutine.
			break
		}

		// Publish the message to the subscribers.
		select {
		case p.publish <- prepared:
		case <-p.ctx.Done():
			return
		}
	}
}

// writePump is the single goroutine for writing messages to the publisher's websocket. It only pings.
func (p *publisher) writePump() {

	// Use the done channel to confirm the closing of the websocket will not interrupt anything.
	defer close(p.done)

	// Create a ping ticker for pinging the publisher in regular intervals.
	pingTicker := time.NewTicker(time.Duration(float64(p.pongDeadline) * pingRatio))
	pingTicker.Stop()

	// Write messages to the subscriber until it is closed.
	var err error
	for {
		select {

		// Ping the publisher when it's time to do so.
		case <-pingTicker.C:
			if err = sendPing(p.conn, p.pongDeadline); err != nil {

				// If the subscriber can't be pinged, close it.
				p.Cancel()
				return
			}

		// Exit this goroutine when the subscriber has closed.
		case <-p.ctx.Done():
			return
		}
	}
}
