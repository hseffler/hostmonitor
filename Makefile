# Makefile for Host Monitor

# Build variables
BINARY_NAME := hostmonitor
VERSION := 1.0.0
BUILD_DIR := build
EXAMPLE_CONFIG_FILE := hostmonitor.yaml.example

# Install variables
INSTALL_DIR := /opt/hostmonitor
CONFIG_DIR := /opt/hostmonitor
SYSTEMD_DIR := /etc/systemd/system
SERVICE_FILE := hostmonitor.service

# Build the binary
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"


# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run the application (for testing)
run:
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) -config $(EXAMPLE_CONFIG_FILE)


# Test the build
test:
	@echo "Testing build..."
	@go test -v
	@echo "Tests complete"

# Install the service
install:
	@echo "Installing $(BINARY_NAME) service..."
	@mkdir -p $(CONFIG_DIR)
	rm -f /var/log/hostmonitor.log /var/log/hostmonitor.error.log
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	cp $(BINARY_NAME).service $(SYSTEMD_DIR)/$(SERVICE_FILE)
	test -f $(CONFIG_DIR)/hostmonitor.yaml || cp $(EXAMPLE_CONFIG_FILE) $(CONFIG_DIR)/hostmonitor.yaml
	chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	systemctl daemon-reload
	systemctl enable $(BINARY_NAME).service
	systemctl start $(BINARY_NAME).service
	@echo "Installation complete. Service started."

# Uninstall the service
uninstall:
	@echo "Uninstalling $(BINARY_NAME) service..."
	systemctl stop $(BINARY_NAME).service 2>/dev/null || true
	systemctl disable $(BINARY_NAME).service 2>/dev/null || true
	rm -f $(SYSTEMD_DIR)/$(SERVICE_FILE)
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	rm -rf $(CONFIG_DIR)
	systemctl daemon-reload
	@echo "Uninstallation complete."

# Show help
help:
	@echo "Makefile for Host Monitor"
	@echo ""
	@echo "Usage:"
	@echo "  make build          - Build the binary"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run            - Run the application (for testing)"
	@echo "  make test           - Run tests"
	@echo "  make install        - Install systemd service"
	@echo "  make uninstall      - Uninstall systemd service"
	@echo "  make help           - Show this help message"

.PHONY: build clean run test help install uninstall
