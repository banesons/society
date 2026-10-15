FROM golang:1.26-bookworm AS builder

WORKDIR /app

# Cache Go modules layer
COPY go.mod go.sum ./
RUN go mod download

# Build statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /chs-api ./cmd/api

# Production distroless image for maximum security and minimal size (~20MB)
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /chs-api /chs-api

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/chs-api"]
