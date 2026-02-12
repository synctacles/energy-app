.PHONY: build test lint clean cross-compile addon-all fmt vet

# Go parameters
BINARY_DIR := dist
GO := go
GOFLAGS := -trimpath
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"

# --- Build ---

build: addon

addon:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/energy-addon ./cmd/energy-addon

# --- Cross-compile (addon for all HA architectures) ---

addon-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/energy-addon-linux-amd64 ./cmd/energy-addon
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/energy-addon-linux-arm64 ./cmd/energy-addon
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/energy-addon-linux-armv7 ./cmd/energy-addon
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/energy-addon-linux-armhf ./cmd/energy-addon
	CGO_ENABLED=0 GOOS=linux GOARCH=386 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/energy-addon-linux-i386 ./cmd/energy-addon

cross-compile: addon-all

# --- Quality ---

test:
	$(GO) test ./... -v -race -count=1

test-short:
	$(GO) test ./... -short -count=1

lint:
	$(GO) vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

fmt:
	gofmt -s -w .

vet:
	$(GO) vet ./...

# --- Clean ---

clean:
	rm -rf $(BINARY_DIR)

# --- Docker ---

docker-addon:
	docker build --build-arg VERSION=$(VERSION) -f deploy/docker/Dockerfile -t local/synctacles-energy:latest .

docker-addon-push: docker-addon
	docker save local/synctacles-energy:latest | gzip > /tmp/synctacles-energy.tar.gz
	scp /tmp/synctacles-energy.tar.gz cc-hub:/tmp/
	ssh cc-hub "scp /tmp/synctacles-energy.tar.gz ha-user:/tmp/"
	ssh cc-hub "ssh ha-user 'sudo docker load < /tmp/synctacles-energy.tar.gz'"

# --- Deps ---

deps:
	$(GO) mod download
	$(GO) mod tidy
