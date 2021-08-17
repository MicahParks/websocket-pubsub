package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MicahParks/websocket-pubsub/env"
	"github.com/MicahParks/websocket-pubsub/pubclient"
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

	// Create the publisher client.
	var pub *pubclient.Publisher
	if pub, _, err = pubclient.New(ctx, u); err != nil {
		logger.Fatalf("Failed to create publisher.\nError: %s", err.Error())
	}

	// Make a channel to catch SIGTERM (Ctrl + C) from the OS.
	ctrlC := make(chan os.Signal, 10)

	// Tell the program to monitor for an interrupt or SIGTERM and report it on the given channel.
	signal.Notify(ctrlC, os.Interrupt, syscall.SIGTERM)

	// Read from stdin and publish all lines.
	go func() {
		stdin := bufio.NewReader(os.Stdin)
		for {
			var line []byte
			if line, _, err = stdin.ReadLine(); err != nil {
				logger.Fatalf("Failed to read stdin.\nError: %s", err.Error())
			}
			if err = pub.Publish(line); err != nil {
				logger.Fatalf("Failed to publish line.\nError: %s", err.Error())
			}
		}
	}()

	// Log events and wait for cancel time.
loop:
	for {
		<-ctrlC
		cancel()
		break loop
	}

	// Wait for the publisher to finish.
	<-pub.Done()
}
