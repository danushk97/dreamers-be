FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p bin && GOOS=linux GOARCH=amd64 go build -v -o ./bin/server ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/bin/server .

EXPOSE 8080

CMD ["./server"]