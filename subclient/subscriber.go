package subclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/MicahParks/pubsub/clients"
)

const (

	// ClientType is the client type to be specified in the "pubsub" HTTP header.
	ClientType = "subscriber"
)

var (

	// ErrUnhandledMessageType indicates that an unhandled websocket message type was received.
	ErrUnhandledMessageType = errors.New("an unhandled websocket message type was received")
)

// Subscriber represents a pubsub subscriber.
type Subscriber struct {
	cancel         context.CancelFunc
	closeOnce      sync.Once
	ctx            context.Context
	conn           *websocket.Conn
	done           chan struct{}
	error          error
	errorMux       sync.RWMutex
	goroutinesDone chan struct{}
	messages       chan []byte
	pongDeadline   time.Duration
}

// New creates a new pubsub.Subscriber. The given ctx is used to close the goroutines launched from this function call.
func New(ctx context.Context, messages chan []byte, u *url.URL, options ...clients.ClientOptions) (subscriber *Subscriber, resp *http.Response, err error) {

	// Flatten the options.
	option := clients.FlattenClientOptions(options)

	// Confirm the messages channel is not nil.
	if messages == nil {
		messages = make(chan []byte)
	}

	// Wrap the context in a cancel function.
	wrappedCtx, cancel := context.WithCancel(ctx)

	// Create the subscriber.
	subscriberStruct := &Subscriber{
		cancel:         cancel,
		ctx:            wrappedCtx,
		done:           make(chan struct{}),
		goroutinesDone: make(chan struct{}),
		messages:       messages,
		pongDeadline:   *option.PongDeadline,
	}

	// Set the correct header to be registered as a subscriber with a name.
	if option.ClientName != "" {
		option.InitialHeaders.Set(clients.ClientNameHeader, option.ClientName)
	}
	option.InitialHeaders.Set(clients.ClientTypeHeader, ClientType)

	// Connect to the subscription server.
	if subscriberStruct.conn, resp, err = option.WebsocketDialer.Dial(u.String(), option.InitialHeaders); err != nil {
		return nil, resp, fmt.Errorf("failed to connect to pubsub server: %w", err)
	}

	// When the server sends the close message, close the client.
	subscriberStruct.conn.SetCloseHandler(func(_ int, _ string) error {
		subscriberStruct.errorMux.Lock()
		subscriberStruct.error = clients.ErrClosedByServer
		subscriberStruct.errorMux.Unlock()
		subscriberStruct.cancel()
		return nil
	})

	// When the context is up for the subscriber, close it.
	go subscriberStruct.close(*option.CloseDeadline)

	// Launch a goroutine to send read messages over the given channel.
	go subscriberStruct.channelMessages()

	return subscriberStruct, resp, nil
}

// Close cancels the subscriber's context, ends its goroutines, and websocket.
func (s *Subscriber) Close() (err error) {
	s.closeOnce.Do(func() {
		s.cancel()
		if err = clients.CloseWebSocket(s.conn, s.pongDeadline); err != nil {
			err = fmt.Errorf("failed to close the subscriber: %w", err)
		}
	})
	return err
}

// Done mimics context.Context's Done method.
func (s *Subscriber) Done() (done <-chan struct{}) {
	return s.done
}

// Error returns the error of why the subscriber closed. It should only be called after the Done method's channel has
// been closed.
func (s *Subscriber) Error() (err error) {
	s.errorMux.RLock()
	defer s.errorMux.RUnlock()
	return s.error
}

// Messages returns the channel to read messages from.
func (s *Subscriber) Messages() (messages <-chan []byte) {
	return s.messages
}

// channelMessages reads messages from the subscription connection and sends them over the messages channel.
func (s *Subscriber) channelMessages() {

	// Use the goroutinesDone channel to confirm the closing of the websocket will not interrupt anything.
	defer close(s.goroutinesDone)

	// Read messages and send them over a channel until the subscriber is closed.
	var messageType int
	var err error
	for {

		// Read the next message.
		var message []byte
		if messageType, message, err = s.conn.ReadMessage(); err != nil {
			var netErr *net.OpError
			if !errors.As(err, &netErr) {
				s.setError(fmt.Errorf("failed to read websocket message: %w", err))
			}
			s.cancel()
			return
		}

		// Only send off binary messages.
		switch messageType {
		case websocket.BinaryMessage:
			select {
			case <-s.ctx.Done():
				return
			case s.messages <- message:
			}
		default:
			s.setError(fmt.Errorf("%w: %d", ErrUnhandledMessageType, messageType))
			s.cancel()
			return
		}
	}
}

// close closes the subscriber when its context ends.
func (s *Subscriber) close(deadline time.Duration) {

	// When the function returns, decrement the wait group.
	defer close(s.done)

	// Wait for the publisher's context to end.
	<-s.ctx.Done()

	// Close the publisher's websocket.
	_ = s.Close() // Ignore any error.

	// Wait for reading goroutines to end.
	select {
	case <-s.goroutinesDone:
	case <-time.After(deadline):
	}
}

// setError sets the error for the subscriber, if it's not already set.
func (s *Subscriber) setError(err error) {
	s.errorMux.Lock()
	defer s.errorMux.Unlock()
	if s.error != nil {
		s.error = err
	}
}
