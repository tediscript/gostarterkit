.PHONY: init tidy build run watch clean help

# Variables
APP_NAME=app
CMD_DIR=cmd/app
BUILD_DIR=bin
GO=go
AIR=air

# Default target
help:
	@echo "Available targets:"
	@echo "  init    - Initialize Go modules"
	@echo "  tidy    - Download and organize dependencies"
	@echo "  build   - Build the application"
	@echo "  run     - Build and run the application"
	@echo "  watch   - Run with hot-reload (requires Air)"
	@echo "  clean   - Clean build artifacts"
	@echo "  help    - Show this help message"

# Initialize Go modules
init:
	$(GO) mod init github.com/tediscript/gostarterkit || true
	$(GO) mod tidy

# Download and organize dependencies
tidy:
	$(GO) mod tidy
	$(GO) mod verify

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build and run the application
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# Run with hot-reload (requires Air)
watch:
	@command -v $(AIR) >/dev/null 2>&1 || { echo "Air is not installed. Install with: go install github.com/cosmtrek/air@latest"; exit 1; }
	$(AIR)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f *.log
	@echo "Clean complete"