BINARY := whooper
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test clean install lint benchmark

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./... -v

test-short:
	go test ./... -short

clean:
	rm -f $(BINARY)
	rm -f coverage.out

install:
	go install $(LDFLAGS) .

lint:
	go vet ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

benchmark:
	go test ./internal/store -bench=. -benchmem -run ^$

release-snapshot:
	goreleaser release --snapshot --clean
