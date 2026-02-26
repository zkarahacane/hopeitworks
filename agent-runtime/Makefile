.PHONY: build test lint clean

BINARY=agent-runtime

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY) .

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
