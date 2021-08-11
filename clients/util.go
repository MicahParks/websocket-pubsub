package clients

import (
	"time"

	"github.com/gorilla/websocket"
)

// CloseWebSocket closes the client's websocket. It return the first error in the chain.
func CloseWebSocket(conn *websocket.Conn, pongDeadline time.Duration) (err error) {

	// This function proceeds even if there are errors. Handle a panic recovery should it occur.
	defer func() {
		_ = recover()
	}()

	// Expect the upcoming close message to be written within a short duration.
	if innerErr := ShortWriteDeadline(conn, pongDeadline); innerErr != nil {
		err = innerErr
	}

	// Write the close message to the client's websocket.
	if innerErr := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); innerErr != nil {
		if err != nil {
			err = innerErr
		}
	}

	// Close the underlying socket for the client's connection.
	if innerErr := conn.Close(); innerErr != nil {
		if err != nil {
			err = innerErr
		}
	}

	return err
}

// ShortWriteDeadline sets a short write deadline relative to the PongDeadline.
func ShortWriteDeadline(conn *websocket.Conn, pongDeadline time.Duration) (err error) {

	// Set the write deadline relative to the pong deadline.
	if err = conn.SetWriteDeadline(time.Now().Add(time.Duration(float64(pongDeadline) * .5))); err != nil { // TODO This duration could be configurable.
		return err
	}

	return nil
}
