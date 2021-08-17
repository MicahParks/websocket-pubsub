package main

import (
	"context"
	"log"

	"github.com/MicahParks/websocket-pubsub/env"
	"github.com/MicahParks/websocket-pubsub/subclient"
)

func main() {

	// Get the pubsub websocket URL from an environment variable.
	pubsubURL, err := env.URL()
	if err != nil {
		log.Fatalf("Failed to get pubsub URL.\nError: %s", err.Error())
	}

	// Create the subscriber.
	var sub *subclient.Subscriber
	if sub, _, err = subclient.New(context.Background(), nil, pubsubURL); err != nil {
		log.Fatalf("Failed to create subscriber.\nError: %s", err.Error())
	}

	// Read all messages and print them to stdout.
	for message := range sub.Messages() {
		log.Printf(`A message was published.\nTopic: %s\nMessage: %s`, pubsubURL.EscapedPath(), string(message))
	}

	// Wait for the subscriber to be completely done.
	<-sub.Done()

	// Print the error of why the subscriber closed.
	if err = sub.Error(); err != nil {
		log.Printf("This error occurred asynchronously in the subscriber: %s", err.Error())
	}
}
