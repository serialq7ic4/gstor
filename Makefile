.PHONY: check test build fmt lint

check:
	./check.sh

test:
	go test ./...

build:
	./build.sh

fmt:
	gofmt -w .

lint:
	golangci-lint run --timeout=5m
