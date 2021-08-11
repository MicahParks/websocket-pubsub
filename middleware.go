package pubsub

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/MicahParks/pubsub/clients"
	"github.com/MicahParks/pubsub/pubclient"
	"github.com/MicahParks/pubsub/subclient"
)

// subscriptionCountdown holds a subscription and a waitgroup for it. When the waitgroup is done, the subscription is
// cancelled.
type subscriptionCountdown struct {
	subscription *subscription
	countdown    sync.WaitGroup
}

// WebSocketHandler creates an HTTP handler via a closure that will upgrade the connection to a websocket and use that
// websocket as either a publisher or a subscriber.
func WebSocketHandler(ctx context.Context, options ...Options) http.HandlerFunc {

	// Flatten the options.
	flatOptions := flattenOptions(options)
	option := *flatOptions.Options
	upgrader := *flatOptions.Upgrader

	// Create a map of paths to subscriptions.
	subscriptMux := sync.Mutex{}
	subscriptions := make(map[string]*subscriptionCountdown)

	// Create the HTTP handler via a closure.
	return func(writer http.ResponseWriter, req *http.Request) {

		// Get the subscription topic.
		subscriptionTopic := req.URL.Path

		// Get the client type from the header.
		isPublisher := false
		switch clientType := req.Header.Get(clients.ClientTypeHeader); clientType {
		case pubclient.ClientType:
			isPublisher = true
		case subclient.ClientType:
			// Keep isPublisher as false.
		default:

			// Report the event.
			if option.Events != nil {
				option.Events <- Event{
					SubscriptionTopic: subscriptionTopic,
					Type:              EventTypeBadHTTPHeaders,
				}
			}

			// Respond with a bad request.
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		// Get the client program name.
		clientName := req.Header.Get(clients.ClientNameHeader)

		// Upgrade the HTTP connection to a web socket.
		conn, err := upgrader.Upgrade(writer, req, nil)
		if err != nil {

			// Report the event.
			if option.Events != nil {
				option.Events <- Event{
					SubscriptionTopic: subscriptionTopic,
					Type:              EventTypeWebsocketUpgradeFailed,
				}
			}

			// Respond with a bad request.
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		// Get or create the subscription in the subscription map.
		subscriptMux.Lock()
		subscriptCountdown, ok := subscriptions[subscriptionTopic]
		if !ok {

			// Create the new subscription.
			subscript := newSubscription(ctx, option)
			subscriptCountdown = &subscriptionCountdown{subscription: subscript}

			// Add the new subscription to the subscription map.
			subscriptions[subscriptionTopic] = subscriptCountdown

			// Make sure the subscription gets cleaned up when there are no clients.
			subscriptCountdown.countdown.Add(1)
			go func() {
				subscriptCountdown.countdown.Wait()
				subscriptCountdown.subscription.cancel()
				subscriptMux.Lock()
				delete(subscriptions, subscriptionTopic)
				subscriptMux.Unlock()
			}()

		} else {

			// Add to the countdown.
			subscriptCountdown.countdown.Add(1)
		}
		subscriptMux.Unlock()
		subscript := subscriptCountdown.subscription

		// When the client's socket is closed, cancel the client and decrement the wait group.
		var client canceller
		clientMux := sync.Mutex{}
		once := sync.Once{}
		conn.SetCloseHandler(func(code int, _ string) error {

			// Only cancel the client and decrement the wait group once.
			once.Do(func() {
				clientMux.Lock()
				client.Cancel()
				clientMux.Unlock()
				subscriptCountdown.countdown.Done()
			})

			// Perform the normal websocket close write.
			message := websocket.FormatCloseMessage(code, "")
			_ = conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second)) // Ignore any error.
			return nil
		})

		// Treat the websocket connection as a publisher or subscriber as instructed.
		if isPublisher {

			// Create the publisher and launch its goroutines.
			pub := newPublisher(subscript.ctx, clientName, conn, subscript.publish, subscriptionTopic, directClientOptions{
				closeDeadline: option.CloseDeadline,
				events:        option.Events,
				pongDeadline:  option.PongDeadline,
			})

			// Report the event.
			if option.Events != nil {
				option.Events <- Event{
					ClientID:          pub.name,
					SubscriptionTopic: subscriptionTopic,
					Type:              EventTypePublisherJoined,
				}
			}

			// Keep track of the new publisher.
			subscript.newPublisher <- pub
			go func() {
				<-pub.ctx.Done()
				select {
				case <-ctx.Done():
				default:
					select {
					case <-ctx.Done():
					case subscript.deletePublisher <- pub:
					}
				}
			}()

			// Keep a reference to the client.
			clientMux.Lock()
			client = pub
			clientMux.Unlock()
		} else {

			// Create the subscriber and launch its goroutines.
			sub := newSubscriber(subscript.ctx, clientName, conn, option.MessageBuffer, subscriptionTopic, directClientOptions{
				closeDeadline: option.CloseDeadline,
				events:        option.Events,
				pongDeadline:  option.PongDeadline,
			})

			// Report the event.
			if option.Events != nil {
				option.Events <- Event{
					ClientID:          sub.name,
					SubscriptionTopic: subscriptionTopic,
					Type:              EventTypeSubscriberJoined,
				}
			}

			// Keep track of the new subscriber.
			subscript.newSubscriber <- sub
			go func() {
				<-sub.ctx.Done()
				select {
				case <-ctx.Done():
				default:
					select {
					case <-ctx.Done():
					case subscript.deleteSubscriber <- sub:
					}
				}
			}()

			// Keep a reference to the client.
			clientMux.Lock()
			client = sub
			clientMux.Unlock()
		}

		// Set the pong handler to expect another pong within the pong deadline.
		conn.SetPongHandler(func(_ string) error {

			// Expect another pong before the new deadline.
			return setReadDeadline(conn, *option.PongDeadline)
		})
	}
}
