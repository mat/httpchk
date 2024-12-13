# Use the official Golang image to create a build artifact.
# This is the build stage.
FROM golang:latest AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .


# Build the Go app
# - CGO_ENABLED=0: Disables CGO, ensuring the build is fully static and does not require C libraries.
# - GOOS=linux: Sets the target operating system to Linux.
# - go build: Compiles the Go application.
# - -a: Forces rebuilding of packages that are already up-to-date.
# - -installsuffix cgo: Adds a suffix to the package installation directory to distinguish it from other builds.
# - -tags netgo: Uses the pure Go implementation of the net package instead of the system's DNS resolver.
# - -ldflags '-s -w': Strips the debug information from the binary to reduce its size.
# - -o httpchk: Specifies the output binary name as 'httpchk'.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -tags netgo -ldflags '-s -w' -o httpchk .

# Start a new stage from scratch
FROM alpine:3.21

WORKDIR /app

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/httpchk /app/httpchk

# Copy templates directory
COPY templates/ templates/
COPY static/ static/

ENV PORT=3000

# Command to run the executable
CMD ["/app/httpchk"]
