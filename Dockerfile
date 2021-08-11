FROM golang:1.16 AS builder

# Create a working directory.
WORKDIR /app

# Get the Golang dependencies for better caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the code in.
COPY . .

# Build the code.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-s -w" -trimpath -o pubsub cmd/server/*


# The actual image being produced.
FROM scratch

# Copy the binary from the builder container.
COPY --from=builder /app/pubsub /pubsub

# Set some default environment variables.
ENV PUBSUB_ADDR="0.0.0.0:8080"

# Start the application.
CMD ["/pubsub"]
