# Start from the latest golang base image
FROM golang:latest as builder
ENV CGO_ENABLED=0

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY *.go .

ENV GOOS=linux
# Build the Go app
RUN for GOARCH in 386 amd64; do go build -a -installsuffix cgo -o cifs-$GOOS-$GOARCH .; done

FROM busybox:1.28.4

# Add Maintainer Info
LABEL maintainer="Gavin Mogan <docker@gavinmogan.com>"

COPY --from=builder /app/cifs-* /

COPY install.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/install.sh

CMD ["/usr/local/bin/install.sh"]
