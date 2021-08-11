package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"time"
)

func main() {

	// Create a logger.
	logger := log.New(os.Stdout, "", 0)

	// Gather information on the size of the test.
	var testDataSize uint
	var maxMessageSize uint
	flag.UintVar(&maxMessageSize, "maxMessageSize", 100000, "The maximum size of a pubsub message in bytes.")
	flag.UintVar(&testDataSize, "testDataSize", 100, "The number of pubsub messages to send.")
	flag.Parse()

	// Create the slice of test data.
	testData := make([]*big.Int, testDataSize)

	// Create some random data.
	rand.Seed(time.Now().UnixNano())
	var err error
	for index := 0; index < int(testDataSize); index++ {

		// Get a random size for the data.
		size := rand.Intn(int(maxMessageSize))

		// Create the random data entry.
		buffer := make([]byte, size)
		if _, err = rand.Read(buffer); err != nil {
			logger.Fatalf("Failed to create random bytes.\nError: %s", err.Error())
		}

		testData[index] = big.NewInt(0).SetBytes(buffer)
	}

	// Sort the data.
	sort.Slice(testData, func(i, j int) bool {
		return testData[i].CmpAbs(testData[j]) == -1
	})

	// gob encode the data.
	buffer := bytes.NewBuffer(nil)
	encoder := gob.NewEncoder(buffer)
	if err = encoder.Encode(testData); err != nil {
		logger.Fatalf("Failed to gob encode test data.\nError: %s", err.Error())
	}

	// Write the test data to a file.
	if err = ioutil.WriteFile("test.dat", buffer.Bytes(), 0666); err != nil {
		logger.Fatalf("Failed to write test data.\nError: %s", err.Error())
	}
}
