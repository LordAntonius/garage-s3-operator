FROM golang:1.25 AS builder
WORKDIR /workspace

# Download modules early to leverage layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the full project and build a static binary.
COPY cmd ./cmd
COPY api ./api
# Build a statically linked binary with symbol stripping to reduce size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -a -installsuffix cgo -ldflags "-s -w" -o /workspace/garage ./cmd/controller


FROM scratch
# Copy the binary
COPY --from=builder /workspace/garage /usr/local/bin/garage
# Copy CA certs from the builder so TLS works in the final image (builder image has them)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/usr/local/bin/garage"]