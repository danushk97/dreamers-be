.PHONY: run test migrate build

run:
	go run ./cmd/server

test:
	go test ./...

build:
	go build -o bin/server ./cmd/server

migrate:
	go run ./cmd/migrate up
