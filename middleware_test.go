package pubsub_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"net/http/httptest"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/MicahParks/websocket-pubsub"
	"github.com/MicahParks/websocket-pubsub/pubclient"
	"github.com/MicahParks/websocket-pubsub/subclient"
)

// TestWebSocketHandler will perform a full test of the server with publisher and subscriber clients using data in
// test.dat generated from the program located in cmd/generate_test_data.
func TestWebSocketHandler(t *testing.T) {

	// Create a logger.
	logger := log.New(os.Stdout, "", 0)

	// Determine the number of publishers and subscribers.
	var publisherCount uint
	var subscriberCount uint
	flag.UintVar(&publisherCount, "publisherCount", 5, "The number of publishers to concurrently to all send the same data at once.")
	flag.UintVar(&subscriberCount, "subscriberCount", 5, "The number of subscribers to each receive a copy of each published message. Each will sort and hash the result to determine if it's as expected.")
	flag.Parse()

	// Read in the test data.
	fileName := "test.dat"
	testData, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read test data file: \"%s\".", fileName)
		t.FailNow()
	}

	// Decode the test data.
	logger.Println("Decoding...")
	decoder := gob.NewDecoder(bytes.NewReader(testData))
	data := make([]*big.Int, 0)
	if err = decoder.Decode(&data); err != nil {
		t.Errorf("Failed to decode test data.\nError: %s", err.Error())
		t.FailNow()
	}
	dataLen := len(data)

	// Compute the expected hash.
	var expected []byte
	var expectedDataLen int
	{

		// Iterate through the test data, append each entry again based on the number of publishers.
		expectedData := make([]*big.Int, dataLen)
		copy(expectedData, data)
		for i := 0; i < int(publisherCount-1); i++ {
			expectedData = append(expectedData, data...)
		}
		expectedDataLen = len(expectedData)

		// Sort the data.
		logger.Printf("Sorting %d entries...", expectedDataLen)
		sort.Slice(expectedData, func(i, j int) bool {
			return expectedData[i].CmpAbs(expectedData[j]) == -1
		})

		// Compute the expected hash.
		logger.Println("Hashing...")
		h := md5.New()
		for _, entry := range expectedData {
			h.Write(entry.Bytes())
		}
		expected = h.Sum(nil)
	}

	// Create a context for the test.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the HTTP test server.
	endpoint := httptest.NewServer(pubsub.WebSocketHandler(ctx, pubsub.Options{
		Options: &pubsub.SubscriptionOptions{
			SubscriberWriteTimeout: &[]time.Duration{time.Minute}[0],
		},
	}))

	// Get the URL for the test server.
	endpointURL := strings.Replace(endpoint.URL, "http", "ws", 1)
	var u *url.URL
	if u, err = url.Parse(endpointURL); err != nil {
		t.Errorf("The httptest server URL was unparsable.\nError: %s", err.Error())
		t.FailNow()
	}

	// Create some subscriber handling assets.
	subDone := &sync.WaitGroup{}

	// Launch the subscribers.
	logger.Println("Starting subscribers...")
	subscribers := make([]*subclient.Subscriber, subscriberCount)
	for i := 0; i < int(subscriberCount); i++ {

		// Create a channel for messages.
		messages := make(chan []byte)

		// Create the subscriber.
		var sub *subclient.Subscriber
		if sub, _, err = subclient.New(ctx, messages, u); err != nil {
			t.Errorf("Failed to create websocket subscriber.\nError: %s", err.Error())
			t.FailNow()
		}

		// Launch a goroutine to handle the publisher.
		subDone.Add(1)
		go handleSubscriber(t, expected, messages, sub, subDone, expectedDataLen)

		// Add the subscriber to the slice of subscribers.
		subscribers[i] = sub
	}

	// Create the publisher handler assets.
	pubDone := &sync.WaitGroup{}
	pubStart := &sync.WaitGroup{}
	pubStart.Add(1)

	// Launch the publishers.
	logger.Println("Starting publishers...")
	publishers := make([]*pubclient.Publisher, publisherCount)
	for i := 0; i < int(publisherCount); i++ {

		// Create the publisher.
		var pub *pubclient.Publisher
		if pub, _, err = pubclient.New(ctx, u); err != nil {
			t.Errorf("Failed to create websocket publisher.\nError: %s", err.Error())
			t.FailNow()
		}

		// Launch a goroutine to handle the publisher.
		pubDone.Add(1)
		go handlePublisher(t, data, pub, pubDone, pubStart)

		// Add the publisher to the slice of publishers.
		publishers[i] = pub
	}

	// Start the publishers.
	pubStart.Done()

	// Wait for the publishing to be over.
	logger.Println("Waiting for publishers to finish...")
	pubDone.Wait()

	// Close the publishers.
	for _, pub := range publishers {
		if err = pub.Close(); err != nil {
			var closeErr *websocket.CloseError
			if !errors.As(err, &closeErr) {
				t.Errorf("Failed to close publisher.\nError: %s", err.Error())
				t.FailNow()
			}
		}
		if err = pub.Error(); err != nil {
			var closeErr *websocket.CloseError
			if !errors.As(err, &closeErr) {
				t.Errorf("Failed to end publisher without error.\nError: %s", err.Error())
				t.FailNow()
			}
		}
	}

	// Wait for the subscribers to finish.
	logger.Println("Waiting for subscribers to finish...")
	subDone.Wait()

	// Close the subscribers.
	for _, sub := range subscribers {
		if err = sub.Close(); err != nil {
			var closeErr *websocket.CloseError
			if !errors.As(err, &closeErr) {
				t.Errorf("Failed to close subscriber.\nError: %s", err.Error())
				t.FailNow()
			}
		}
		if err = sub.Error(); err != nil {
			var closeErr *websocket.CloseError
			if !errors.As(err, &closeErr) {
				t.Errorf("Failed to end subscriber without error.\nError: %s", err.Error())
				t.FailNow()
			}
		}
	}

	logger.Println("Done.")
}

// handlePublisher is a separate goroutine to manage a publisher client.
func handlePublisher(t *testing.T, data []*big.Int, pub *pubclient.Publisher, pubDone *sync.WaitGroup, pubStart *sync.WaitGroup) {

	// Wait for the publishing to start.
	pubStart.Wait()

	// Decrement the done wait group.
	defer pubDone.Done()

	// Write messages until the context expires.
	var err error
	for _, entry := range data {

		// Publish the data entry.
		if err = pub.Publish(entry.Bytes()); err != nil {
			t.Errorf("Failed to publish message.\nError: %s", err.Error())
		}
	}
}

// handleSubscriber is a separate goroutine to manage a subscriber client.
func handleSubscriber(t *testing.T, expected []byte, messages <-chan []byte, sub *subclient.Subscriber, subDone *sync.WaitGroup, testDataSize int) {

	// Decrement the done wait group when finished.
	defer subDone.Done()

	// Read all messages from the channel.
	receivedData := make([]*big.Int, testDataSize)
	index := 0
loop:
	for {
		select {
		case <-sub.Done():
			if err := sub.Error(); err != nil {
				t.Errorf("Client was closed unexpectedly.\nError: %s", err.Error())
				return
			}
		case message := <-messages:
			receivedData[index] = big.NewInt(0).SetBytes(message)
			index++
			if index >= testDataSize {
				break loop
			}
		}
	}

	// Sort the data.
	sort.Slice(receivedData, func(i, j int) bool {
		return receivedData[i].CmpAbs(receivedData[j]) == -1
	})

	// Hash the data.
	h := md5.New()
	for _, data := range receivedData {
		h.Write(data.Bytes())
	}

	// Confirm thi data is as expected.
	if !bytes.Equal(h.Sum(nil), expected) {
		t.Errorf("Subscriber data checksum did not match.")
	}
}
