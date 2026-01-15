.PHONY: lint test install clean release

GO ?= go
GOLANGCI_LINT = golangci-lint

# Find all Go source files
SOURCES := $(shell find . -name "*.go")

# Binary name
BINARY := aver

# Build target
$(BINARY): $(SOURCES)
	$(GO) build -o $@ ./cmd/aver

lint:
	$(GOLANGCI_LINT) run ./...

test:
	$(GO) test ./...

install: $(BINARY)
	$(GO) install ./cmd/aver

clean:
	rm -f $(BINARY)

release:
	@./scripts/release.sh

# Default target
all: $(BINARY)