.PHONY: build test vet run

build:
	go build -o bin/agent ./cmd/agent

test:
	go test ./...

vet:
	go vet ./...

run:
	go run ./cmd/agent -prompt "What is (12.5*3 - 4) / 2?"
