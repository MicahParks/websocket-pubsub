[![Go Report Card](https://goreportcard.com/badge/github.com/MicahParks/pubsub)](https://goreportcard.com/report/github.com/MicahParks/pubsub) [![Go Reference](https://pkg.go.dev/badge/github.com/MicahParks/pubsub.svg)](https://pkg.go.dev/github.com/MicahParks/pubsub)

# pubsub

## Design

This is an HTTP websocket publish-subscribe server written in Golang with publisher and subscriber client packages. Each
different URL path is its own subscription topic.

All assets for this project are stored in memory. This means the pubsub server
[scales vertically](https://stackoverflow.com/a/11715598/14797322) with its sole host's processing power, memory, and
network.

The primary use case for this pubsub server is smaller projects where it makes sense to separate publisher and
subscriber logic into different programs or hosts/containers.

This pubsub server is not recommended for larger projects that need to scale a pubsub server past the resources
available on a single host.

The websocket package used is [`github.com/gorilla/websocket`](https://github.com/gorilla/websocket).

## Deploying the server

For Dockerized deployments the pre-built image is available
at [on DockerHub](https://hub.docker.com/repository/docker/micahparks/pubsub) using this path: `micahparks/pubsub`. The
`Dockerfile` is also included in the root directory of this project.

Environment variables:

|Name         |Description                                                                                        |
|-------------|---------------------------------------------------------------------------------------------------|
|`PUBSUB_ADDR`|The `addr` argument to pass to [`http.ListenAndServe`](https://pkg.go.dev/net/http#ListenAndServe).|

## Client usage

Clients exclusively communicate through
[HTTP websocket binary messages](https://datatracker.ietf.org/doc/html/rfc6455#section-5.6). This means any encoding is
allowed: JSON, protobuf, gob, etc. The Go datatype used is `[]byte`.

The examples below do not cover some more advanced use cases. Using the `clients.Options` data structure, a client name
can be specified, as well as custom websocket dialer or initial headers to the dialing request. Each client is assigned
a UUID on the pubsub server side to uniquely identify clients in the logs.

Full publisher example:
```go
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
```

Full subscriber example:
```go
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
```

## Testing

### Generating the test data
In order to test, the test data must be generated.

```bash
go run cmd/generate_test_data/main.go
```

This will create a file called `test.dat` in the project's root directory, which is read by the test.

The test data are constructed from a random number of bytes up to the `maxMessageSize` flag. That number of bytes is the
quantity of random bytes read from a seeded `math/rand`. Those random bytes are interpreted as an unsigned integer,
`*big/Int`, then `gob` encoded.

*Note:*

* The test data's message quantity and maximum message size is configurable through flags.

### Performing a test
After the test data, `test.dat`, has been generated, a full functional test can be performed.

```bash
go test -cover -race
```

The test will read in the test data, sort it, hash it, then spin up publishers and subscribers.

The publishers will get a reference to the test data, and send it message-by-message, until each publisher has sent
every message.

The subscribers will read a copy of every message published, sort all the messages, hash them, then confirm the hash
with what's expected.

The amount of memory this test consumes scales greatly with:
* The number of messages.
* The maximum size of messages.
* The number of publishers.
* The number of subscribers.

*Note:*

* This test pretty much keeps _everything_ in memory until the test is over. This includes a bunch of copies of the test
  data. Be careful not to run out of memory. The defaults should be safe for most machines.
* The number of publishers and subscribers is configurable through flags.

### Test coverage
This one test covers 76.4% of the code. Additional tests are welcome in contributions.
