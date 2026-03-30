.PHONY: run test migrate build mockgen

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

MOCKGEN_VERSION := v1.6.0
MOCKGEN_BIN := ./bin/tools/mockgen

mockgen:
	@mkdir -p ./internal/mocks ./bin/tools
	@if [ ! -x "$(MOCKGEN_BIN)" ]; then GOSUMDB=off GOBIN=$(CURDIR)/bin/tools go install github.com/golang/mock/mockgen@$(MOCKGEN_VERSION); fi
	@$(MOCKGEN_BIN) -source=internal/domain/player/repository.go -destination=internal/mocks/mock_player_repository.go -package=mocks
	@$(MOCKGEN_BIN) -source=internal/domain/storage/file_uploader.go -destination=internal/mocks/mock_storage_interfaces.go -package=mocks
