# Dockerfile for the Synctacles Energy central price server.
# Produces a minimal static binary.

FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /energy-server ./cmd/energy-server

# --- Runtime ---
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /energy-server /usr/bin/energy-server

EXPOSE 8080
ENTRYPOINT ["/usr/bin/energy-server"]
