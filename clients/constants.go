package clients

import (
	"time"
)

const (

	// ClientTypeHeader is the HTTP header where the client type resides.
	ClientTypeHeader = "pubsub"

	// ClientNameHeader is the HTTP header where the client's program name is stored.
	ClientNameHeader = "pubsub-client-name"

	// CloseDeadline is the time to wait to gracefully closing something. After that, it's closed regardless.
	CloseDeadline = time.Second * 5

	// PongDeadline is the number of seconds to wait before the next pong.
	PongDeadline = time.Second * 5
)
