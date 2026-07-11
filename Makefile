VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS := -ldflags "-X github.com/hyper0x/icloud-pull/internal/cli.Version=$(VERSION) -X github.com/hyper0x/icloud-pull/internal/cli.Commit=$(COMMIT)"

.PHONY: build install clean test lint vet

build:
	go build $(LDFLAGS) -o icloud-pull .

install:
	go install $(LDFLAGS) .

clean:
	rm -f icloud-pull

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
