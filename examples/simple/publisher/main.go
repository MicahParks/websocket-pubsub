package main

import (
	"context"
	"log"

	"github.com/MicahParks/pubsub/env"
	"github.com/MicahParks/pubsub/pubclient"
)

func main() {

	// Get the pubsub websocket URL from an environment variable.
	pubsubURL, err := env.URL()
	if err != nil {
		log.Fatalf("Failed to get pubsub URL.\nError: %s", err.Error())
	}

	// Create the publisher.
	var pub *pubclient.Publisher
	if pub, _, err = pubclient.New(context.Background(), pubsubURL); err != nil {
		log.Fatalf("Failed to create publisher.\nError: %s", err.Error())
	}

	// Publish a message.
	if err = pub.Publish([]byte("message")); err != nil {
		log.Fatalf("Failed to publish message.\nError: %s", err.Error())
	}

	// Close the publisher.
	if err = pub.Close(); err != nil {
		log.Fatalf("Failed to close the publisher.\nError: %s", err.Error())
	}

	// Wait for the publisher to close completely.
	<-pub.Done()

	// Print an asynchronous error for the publisher, if any.
	if err = pub.Error(); err != nil {
		log.Printf("This error occurred asynchronously in the publisher: %s", err.Error())
	}
}
