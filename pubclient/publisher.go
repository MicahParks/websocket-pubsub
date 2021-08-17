package pubclient

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

	"github.com/MicahParks/websocket-pubsub/clients"
)

const (

	// ClientType is the client type to be specified in the "pubsub" HTTP header.
	ClientType = "publisher"
)

// Publisher represents a pubsub publisher.
type Publisher struct {
	cancel         context.CancelFunc
	closeOnce      sync.Once
	ctx            context.Context
	conn           *websocket.Conn
	done           chan struct{}
	error          error
	errorMux       sync.RWMutex
	goroutinesDone chan struct{}
	pongDeadline   time.Duration
}

// New creates a new Publisher. The given ctx is used to close the goroutines launched from this function call.
func New(ctx context.Context, u *url.URL, options ...clients.Options) (publisher *Publisher, resp *http.Response, err error) {

	// Flatten the options.
	option := clients.FlattenClientOptions(options)

	// Wrap the context in a cancel function.
	wrappedCtx, cancel := context.WithCancel(ctx)

	// Create the publisher.
	publisherStruct := &Publisher{
		cancel:         cancel,
		ctx:            wrappedCtx,
		done:           make(chan struct{}),
		goroutinesDone: make(chan struct{}),
		pongDeadline:   *option.PongDeadline,
	}

	// Set the correct header to be registered as a publisher with a name.
	if option.ClientName != "" {
		option.InitialHeaders.Set(clients.ClientNameHeader, option.ClientName)
	}
	option.InitialHeaders.Set(clients.ClientTypeHeader, ClientType)

	// Connect to the subscription server.
	if publisherStruct.conn, resp, err = option.WebsocketDialer.Dial(u.String(), option.InitialHeaders); err != nil {
		return nil, resp, fmt.Errorf("failed to connect to pubsub server: %w", err)
	}

	// When the server sends the close message, close the client.
	publisherStruct.conn.SetCloseHandler(func(_ int, _ string) error {
		publisherStruct.errorMux.Lock()
		publisherStruct.error = clients.ErrClosedByServer
		publisherStruct.errorMux.Unlock()
		publisherStruct.cancel()
		return nil
	})

	// When the context is up for the subscriber, close it.
	go publisherStruct.close(*option.CloseDeadline)

	// Ignore messages from the subscription server.
	go publisherStruct.ignoreServer()

	return publisherStruct, resp, nil
}

// Close cancels the publisher's context, ends its goroutines, and websocket.
func (p *Publisher) Close() (err error) {
	p.closeOnce.Do(func() {
		p.cancel()
		if err = clients.CloseWebSocket(p.conn, p.pongDeadline); err != nil {
			err = fmt.Errorf("failed to close the subscriber: %w", err)
		}
	})
	return err
}

// Done mimics context.Context's Done method.
func (p *Publisher) Done() (done <-chan struct{}) {
	return p.done
}

// Error returns the error of why the subscriber closed. It should only be called after the Done method's channel has
// been closed.
func (p *Publisher) Error() (err error) {
	p.errorMux.RLock()
	defer p.errorMux.RUnlock()
	return p.error
}

// Publish publishes the message to the subscription. This method is not safe to call asynchronously. A pumping
// goroutine is recommended.
func (p *Publisher) Publish(message []byte) (err error) {
	if err = p.conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
		return err
	}
	return nil
}

// close closes the publisher when its context ends.
func (p *Publisher) close(deadline time.Duration) {

	// When the function returns, decrement the wait group.
	defer close(p.done)

	// Wait for the publisher's context to end.
	<-p.ctx.Done()

	// Close the publisher's websocket.
	_ = p.Close() // Ignore any error.

	// Wait for reading goroutines to end.
	select {
	case <-p.goroutinesDone:
	case <-time.After(deadline):
	}
}

// ignoreServer is a required goroutine to read messages from the subscription server. All messages are ignored.
func (p *Publisher) ignoreServer() {

	// Use the goroutinesDone channel to confirm the closing of the websocket will not interrupt anything.
	defer close(p.goroutinesDone)

	// Read messages from the server to clear them out until the publisher is closed.
	var err error
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			_, _, err = p.conn.ReadMessage()
			if err != nil {
				var netErr *net.OpError
				if !errors.As(err, &netErr) { // TODO Remove this?
					p.setError(err)
				}
				p.cancel()
				return
			}
		}
	}
}

// setError sets the error for the publisher, if it's not already set.
func (p *Publisher) setError(err error) {
	p.errorMux.Lock()
	defer p.errorMux.Unlock()
	if p.error != nil {
		p.error = err
	}
}
