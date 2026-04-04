# Multi-stage build for HA addon.
# Stage 1: Build Go binary (CGO_ENABLED=0 for static linking).
# Stage 2: Minimal Alpine base (HA requires bashio for addon config).

# Global ARG — must be before first FROM to be available in later FROM.
ARG BUILD_FROM=ghcr.io/home-assistant/amd64-base:3.18

# --- Build stage ---
FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /energy-addon ./cmd/energy-addon

# --- Runtime stage ---
# HA addons need bashio for reading addon options from /data/options.json.
FROM ${BUILD_FROM}

COPY --from=builder /energy-addon /usr/bin/energy-addon
COPY deploy/docker/run.sh /

RUN chmod +x /run.sh /usr/bin/energy-addon

CMD ["/run.sh"]
