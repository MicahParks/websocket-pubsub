package pubsub

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"

	"github.com/MicahParks/websocket-pubsub/clients"
)

// subscriber holds all the required information for a subscriber to the pubsub system.
type subscriber struct {
	cancel       context.CancelFunc
	conn         *websocket.Conn
	ctx          context.Context
	done         chan struct{}
	events       chan<- Event
	name         string
	pongDeadline time.Duration
	send         chan *websocket.PreparedMessage
	topic        string
}

// newSubscriber creates a new subscriber and launches the appropriate goroutines.
func newSubscriber(ctx context.Context, name string, conn *websocket.Conn, messageBuffer uint, topic string, options ...directClientOptions) (sub *subscriber) {

	// Wrap the context in a cancel function.
	subCtx, cancel := context.WithCancel(ctx)

	// Flatten the options.
	option := flattenDirectClientOptions(options)

	// Create the subscriber.
	sub = &subscriber{
		cancel:       cancel,
		conn:         conn,
		ctx:          subCtx,
		done:         make(chan struct{}),
		events:       option.events,
		name:         name + "-" + uuid.NewV4().String(),
		pongDeadline: *option.pongDeadline,
		send:         make(chan *websocket.PreparedMessage, messageBuffer),
		topic:        topic,
	}

	// Launch the subscriber pumping goroutines.
	go sub.readPump()
	go sub.writePump()

	// Close the client when the context expires.
	go func() {

		// Report the event when this function returns.
		if sub.events != nil {
			event := Event{
				ClientID:          sub.name,
				SubscriptionTopic: sub.topic,
				Type:              EventTypeSubscriberLeft,
			}
			defer func() {
				select {
				case sub.events <- event:
				case <-time.After(*option.closeDeadline):
				}
			}()
		}

		// Wait for the client's context to expire.
		<-subCtx.Done()

		// Wait for the writer goroutines to exit.
		select {
		case <-sub.done:
		case <-time.After(*option.closeDeadline):
		}

		// Close the subscriber's websocket.
		_ = clients.CloseWebSocket(sub.conn, sub.pongDeadline) // Ignore any error.
	}()

	return sub
}

// Cancel implements the canceller interface.
func (s *subscriber) Cancel() {
	s.cancel()
}

// readPump is the single goroutine for reading websocket messages from the subscriber. It handles pings and pongs.
func (s *subscriber) readPump() {

	// If this goroutine is the first to exit, close the publisher.
	defer s.Cancel()

	// Subscribers shouldn't send any messages except ping/pong. Read these messages, which will be taken care of by
	// their handlers.
	var err error
	for {

		// Read the next message from the client. To allow handlers to happen. Ignore non handled messages.
		if _, _, err = s.conn.ReadMessage(); err != nil {

			// End the goroutine.
			break
		}
	}
}

// writePump is the single goroutine for writing messages to the subscriber's websocket. It handles pinging the client,
// writing subscription messages, and closing the client.
func (s *subscriber) writePump() {

	// Use the done channel to confirm the closing of the websocket will not interrupt anything.
	defer close(s.done)

	// Create a ping ticker for pinging the subscriber in regular intervals.
	pingTicker := time.NewTicker(time.Duration(float64(s.pongDeadline) * pingRatio))
	pingTicker.Stop()

	// Write messages to the subscriber until it is closed.
	var err error
	for {
		select {

		// Ping the subscriber when it's time to do so.
		case <-pingTicker.C:
			if err = sendPing(s.conn, s.pongDeadline); err != nil {

				// If the subscriber can't be pinged, close it.
				s.Cancel()
				return
			}

		// Send subscription messages to the subscriber when there are any.
		case prepared := <-s.send:

			// Expect the upcoming message to be written within a short duration.
			if err = clients.ShortWriteDeadline(s.conn, s.pongDeadline); err != nil {
				return
			}

			// Write the message to the subscriber's websocket.
			if err = s.conn.WritePreparedMessage(prepared); err != nil {

				// If the subscriber can't receive messages, close it.
				s.Cancel()
				return
			}

		// Exit this goroutine when the subscriber has closed.
		case <-s.ctx.Done():
			return
		}
	}
}
