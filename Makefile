.PHONY: run test migrate build

# Use external linker on macOS to avoid dyld "missing LC_UUID" (Go issue #68678).
# Requires Xcode Command Line Tools. Alternatively, upgrade to Go 1.24+.
LDFLAGS := -ldflags="-linkmode=external"

run:
	go run $(LDFLAGS) ./cmd/server

test:
	go test ./...

# .PHONY: go-build-api ## Build the binary file for API server
go-build-api:
	@CGO_ENABLED=0 GOOS=$(UNAME_OS) GOARCH=$(UNAME_ARCH) go build -v -o ./bin/server ./cmd/server

migrate:
	go run ./cmd/migrate up
