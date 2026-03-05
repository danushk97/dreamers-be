# -----------------------------
# Builder Stage
# -----------------------------
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache make

# Copy go mod first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build using Makefile
RUN make go-build-api


# -----------------------------
# Runtime Stage (small image)
# -----------------------------
FROM alpine:3.19

# RUN apk add --no-cache make ca-certificates

WORKDIR /app

# Copy compiled binary only
COPY --from=builder /app/bin/server ./server

EXPOSE 8080

CMD ["./server"]