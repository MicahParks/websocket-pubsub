package clients

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Options represents information used to connect a publisher or subscriber to a subscription that already has
// default values.
type Options struct {

	// ClientName is the name of the client's program. It is used in the ClientNameHeader HTTP header.
	ClientName string

	// CloseDeadline is the time to wait to gracefully closing something. After that, it's closed regardless.
	CloseDeadline *time.Duration

	// InitialHeaders are the headers send during the first request to create a websocket.
	InitialHeaders http.Header

	// PongDeadline is the websocket write deadline to set when expecting a response.
	PongDeadline *time.Duration

	// WebsocketDialer is the *websocket.Dialer to use, if the default isn't sufficient.
	WebsocketDialer *websocket.Dialer
}

// FlattenClientOptions takes in a slice of Options, uses the highest index of their fields' values to create one
// Options.
func FlattenClientOptions(options []Options) (option Options) {
	for _, opt := range options {
		if opt.ClientName != "" {
			option.ClientName = opt.ClientName
		}
		if opt.CloseDeadline != nil {
			option.CloseDeadline = opt.CloseDeadline
		}
		if opt.InitialHeaders != nil {
			option.InitialHeaders = opt.InitialHeaders
		}
		if opt.PongDeadline != nil {
			option.PongDeadline = opt.PongDeadline
		}
		if opt.WebsocketDialer != nil {
			option.WebsocketDialer = opt.WebsocketDialer
		}
	}
	if option.CloseDeadline == nil {
		deadline := CloseDeadline
		option.CloseDeadline = &deadline
	}
	if option.InitialHeaders == nil {
		option.InitialHeaders = http.Header{}
	}
	if option.PongDeadline == nil {
		deadline := PongDeadline
		option.PongDeadline = &deadline
	}
	if option.WebsocketDialer == nil {
		option.WebsocketDialer = websocket.DefaultDialer
	}
	return option
}
