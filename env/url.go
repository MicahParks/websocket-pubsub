package env

import (
	"fmt"
	"net/url"
	"os"
)

// URL returns the value of the PUBSUB_ADDR as a *url.URL.
func URL() (u *url.URL, err error) {

	// Get the pubsub server address.
	addr := os.Getenv("PUBSUB_ADDR")

	// Parse the addr as a URL.
	if u, err = url.Parse(addr); err != nil {
		return nil, fmt.Errorf("failed to parse PUBSUB_ADDR environment variable as a URL: %w", err)
	}

	return u, nil
}
