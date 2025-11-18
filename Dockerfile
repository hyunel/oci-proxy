FROM golang:1.25.1-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o oci-proxy ./cmd/oci-proxy

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/oci-proxy .
COPY config.yaml .

RUN mkdir -p /tmp/oci-proxy-cache && \
    chmod 755 /tmp/oci-proxy-cache

EXPOSE 80

ENTRYPOINT ["/app/oci-proxy"]
CMD ["-c", "/app/config.yaml"]
