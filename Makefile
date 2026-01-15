.PHONY: lint test install clean

GO = /Users/llimllib/.local/share/mise/installs/go/1.25.5/bin/go
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

# Default target
all: $(BINARY)