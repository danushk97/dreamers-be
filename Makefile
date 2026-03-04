.PHONY: run test migrate build

# Use external linker on macOS to avoid dyld "missing LC_UUID" (Go issue #68678).
# Requires Xcode Command Line Tools. Alternatively, upgrade to Go 1.24+.
LDFLAGS := -ldflags="-linkmode=external"

run:
	go run $(LDFLAGS) ./cmd/server

test:
	go test ./...

build:
	go build $(LDFLAGS) -o bin/server ./cmd/server

migrate:
	go run ./cmd/migrate up
