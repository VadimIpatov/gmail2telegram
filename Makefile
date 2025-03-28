.PHONY: build run test clean lint deps help token

# Build variables
BINARY_NAME=gmail2telegram
GO=go
GOBUILD=$(GO) build
GOTEST=$(GO) test

# Build the application
build:
	$(GOBUILD) -o ./$(BINARY_NAME) ./src

# Run the application
run:
	$(GO) run ./src

# Run tests with coverage
test:
	$(GOTEST) -v -coverprofile=coverage.txt ./src/...

# Clean build artifacts
clean:
	rm -f ./$(BINARY_NAME)
	rm -f ./coverage.txt

# Run linter
lint:
	go tool mvdan.cc/gofumpt -l -w ./src  
	go tool github.com/golangci/golangci-lint/cmd/golangci-lint run

# Install dependencies
deps:
	$(GO) mod tidy

# Generate Gmail token (requires credentials.json)
token:
	$(GO) run ./src --generate-token

# Help command
help:
	@echo "Available commands:"
	@echo "  make build    - Build the application"
	@echo "  make run      - Run the application"
	@echo "  make test     - Run tests with coverage"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make lint     - Run linter"
	@echo "  make deps     - Install dependencies"
	@echo "  make token    - Generate Gmail token"
	@echo "  make help     - Show this help message" 