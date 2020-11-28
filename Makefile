GOARCH=amd64

build:
	go build -ldflags="-s -w" -o bin/server-linux-${GOARCH} ./cmd/server/main.go
	go build -ldflags="-s -w" -o bin/client-linux-${GOARCH} ./cmd/client/main.go

lint:
	golangci-lint run ./cmd/... ./internal/...

deps:
	go mod verify && \
	go mod tidy && \
	go mod vendor

.PHONY: pkg build lint deps
