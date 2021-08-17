package pubsub

import (
	"time"

	"github.com/gorilla/websocket"

	"github.com/MicahParks/websocket-pubsub/clients"
)

const (

	// pingRatio is the ratio of when pings are sent to when pongs are expected.
	pingRatio = .7 // TODO Could make constant configurable.
)

// sendPing sends a ping to the client's websocket.
func sendPing(conn *websocket.Conn, pongDeadline time.Duration) (err error) {

	// Expect the upcoming ping to be written within a short duration.
	if err = clients.ShortWriteDeadline(conn, pongDeadline); err != nil {
		return err
	}

	// Send the ping.
	if innerErr := conn.WriteMessage(websocket.PingMessage, nil); innerErr != nil {
		return err
	}

	return nil
}

// setReadDeadline sets a read deadline equal to the PongDeadline.
func setReadDeadline(conn *websocket.Conn, pongDeadline time.Duration) (err error) {

	// Set the read deadline equal to the pong deadline.
	if err = conn.SetReadDeadline(time.Now().Add(pongDeadline)); err != nil {
		return err
	}

	return nil
}
