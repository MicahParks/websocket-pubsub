package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MicahParks/pubsub/env"
	"github.com/MicahParks/pubsub/subclient"
)

func main() {

	// Create a logger.
	logger := log.New(os.Stdout, "", 0)

	// Get the pubsub server address.
	u, err := env.URL()
	if err != nil {
		logger.Fatalf("Failed to connect to pubsub server.\nError: %s", err.Error())
	}
	logger.Printf("Connecting to: \"%s\".", u.String())

	// Create a context for the publisher.
	ctx, cancel := context.WithCancel(context.Background())

	// Create the subscriber client.
	var sub *subclient.Subscriber
	if sub, _, err = subclient.New(ctx, nil, u); err != nil {
		logger.Fatalf("Failed to create subscriber.\nError: %s", err.Error())
	}

	// Make a channel to catch SIGTERM (Ctrl + C) from the OS.
	ctrlC := make(chan os.Signal, 10)

	// Tell the program to monitor for an interrupt or SIGTERM and report it on the given channel.
	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	// Print messages as they come in.
loop:
	for {
		select {
		case message := <-sub.Messages():
			logger.Println(string(message))
		case <-ctrlC:
			cancel()
			break loop
		}
	}

	// Wait for the subscriber to finish.
	<-sub.Done()
}
