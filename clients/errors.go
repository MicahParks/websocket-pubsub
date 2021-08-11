package clients

import (
	"errors"
)

var (

	// ErrClosedByServer indicates that the client was closed by the server.
	ErrClosedByServer = errors.New("the client was closed by the server")
)
