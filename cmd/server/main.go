package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/MicahParks/websocket-pubsub"
)

func main() {

	// Create a logger.
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Get the address to listen on.
	addr := os.Getenv("PUBSUB_ADDR")
	logger.Printf("Listening on: \"%s\".\n", addr)

	// Create a context for the server.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the websocket pubsub handler.
	events := make(chan pubsub.Event)
	subscript := pubsub.WebSocketHandler(ctx, pubsub.Options{Options: &pubsub.SubscriptionOptions{
		Events: events,
	}})
	http.HandleFunc("/", subscript)

	// Make a channel to catch SIGTERM (Ctrl + C) from the OS.
	ctrlC := make(chan os.Signal, 10)

	// Tell the program to monitor for an interrupt or SIGTERM and report it on the given channel.
	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	// Start the server and end when the server is finished.
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Fatalf("Failed to serve pubsub server.\nError: %s", err.Error())
		}
	}()

	// Log events and wait for cancel time.
loop:
	for {
		select {
		case <-ctrlC:
			cancel()
			break loop
		case event := <-events:
			logEvent(logger, event)
		}
	}
}

// logEvent logs the event in JSON format.
func logEvent(logger *log.Logger, event pubsub.Event) {

	// Marshal the event to JSON.
	printMe, err := json.MarshalIndent(event, "", "    ")
	if err != nil {
		logger.Fatalf("Failed to marshal event to JSON.\nError: %s", err.Error())
	}

	// Log the event.
	logger.Printf("\n%s", string(printMe))
}
