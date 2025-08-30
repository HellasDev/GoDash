# GoDash - Terminal Productivity Dashboard
# Makefile for building and installing on Linux systems

# Variables
BINARY_NAME := godash
INSTALL_PREFIX := /usr/local
BINARY_DIR := $(INSTALL_PREFIX)/bin
DESKTOP_DIR := $(INSTALL_PREFIX)/share/applications
ICON_DIR := $(INSTALL_PREFIX)/share/pixmaps
DOCS_DIR := $(INSTALL_PREFIX)/share/doc/$(BINARY_NAME)
MAN_DIR := $(INSTALL_PREFIX)/share/man/man1

# Go build variables
GO := go
GOFLAGS := -ldflags="-s -w" -trimpath

# Default target
all: build

# Build the application
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .
	@echo "✅ Build complete!"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	$(GO) clean
	@echo "✅ Clean complete!"

# Install the application (requires root)
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	
	# Install binary
	install -Dm755 $(BINARY_NAME) $(DESTDIR)$(BINARY_DIR)/$(BINARY_NAME)
	
	# Install desktop entry
	install -Dm644 $(BINARY_NAME).desktop $(DESTDIR)$(DESKTOP_DIR)/$(BINARY_NAME).desktop
	
	# Install icon
	install -Dm644 logo.png $(DESTDIR)$(ICON_DIR)/$(BINARY_NAME).png
	
	# Install man page
	install -Dm644 $(BINARY_NAME).1 $(DESTDIR)$(MAN_DIR)/$(BINARY_NAME).1
	
	# Install documentation
	install -Dm644 README.md $(DESTDIR)$(DOCS_DIR)/README.md
	install -Dm644 LICENSE $(DESTDIR)$(DOCS_DIR)/LICENSE
	
	@echo "✅ Installation complete!"
	@echo "💡 Run 'godash' to start the application"

# Uninstall the application (requires root)
uninstall:
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	
	# Remove binary
	rm -f $(DESTDIR)$(BINARY_DIR)/$(BINARY_NAME)
	
	# Remove desktop entry
	rm -f $(DESTDIR)$(DESKTOP_DIR)/$(BINARY_NAME).desktop
	
	# Remove icon
	rm -f $(DESTDIR)$(ICON_DIR)/$(BINARY_NAME).png
	
	# Remove man page
	rm -f $(DESTDIR)$(MAN_DIR)/$(BINARY_NAME).1
	
	# Remove documentation
	rm -rf $(DESTDIR)$(DOCS_DIR)
	
	@echo "✅ Uninstallation complete!"
	@echo "💡 User data in ~/.config/GoDash and ~/.local/share/GoDash remains intact"

# Install for current user only (no root required)
install-user: build
	@echo "👤 Installing $(BINARY_NAME) for current user..."
	
	# Create directories
	mkdir -p ~/.local/bin
	mkdir -p ~/.local/share/applications
	mkdir -p ~/.local/share/doc/$(BINARY_NAME)
	mkdir -p ~/.local/share/man/man1
	mkdir -p ~/.local/share/pixmaps
	
	# Install binary
	install -m755 $(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)
	
	# Install desktop entry
	install -m644 $(BINARY_NAME).desktop ~/.local/share/applications/$(BINARY_NAME).desktop
	
	# Install icon
	install -m644 logo.png ~/.local/share/pixmaps/$(BINARY_NAME).png
	
	# Install man page
	install -m644 $(BINARY_NAME).1 ~/.local/share/man/man1/$(BINARY_NAME).1
	
	# Install documentation
	install -m644 README.md ~/.local/share/doc/$(BINARY_NAME)/README.md
	install -m644 LICENSE ~/.local/share/doc/$(BINARY_NAME)/LICENSE
	
	@echo "✅ User installation complete!"
	@echo "💡 Make sure ~/.local/bin is in your PATH"
	@echo "💡 Run 'godash' to start the application"

# Uninstall for current user
uninstall-user:
	@echo "👤 Uninstalling $(BINARY_NAME) for current user..."
	
	# Remove binary
	rm -f ~/.local/bin/$(BINARY_NAME)
	
	# Remove desktop entry
	rm -f ~/.local/share/applications/$(BINARY_NAME).desktop
	
	# Remove icon
	rm -f ~/.local/share/pixmaps/$(BINARY_NAME).png
	
	# Remove man page
	rm -f ~/.local/share/man/man1/$(BINARY_NAME).1
	
	# Remove documentation
	rm -rf ~/.local/share/doc/$(BINARY_NAME)
	
	@echo "✅ User uninstallation complete!"

# Development targets
dev: build
	@echo "🚀 Starting $(BINARY_NAME) in development mode..."
	./$(BINARY_NAME)

# Run tests
test:
	@echo "🧪 Running tests..."
	$(GO) test -v ./...

# Update dependencies
deps:
	@echo "📦 Updating dependencies..."
	$(GO) mod tidy
	$(GO) mod download

# Show help
help:
	@echo "GoDash - Terminal Productivity Dashboard"
	@echo ""
	@echo "Available targets:"
	@echo "  build         Build the application"
	@echo "  clean         Clean build artifacts"
	@echo "  install       Install system-wide (requires root)"
	@echo "  uninstall     Uninstall system-wide (requires root)"
	@echo "  install-user  Install for current user only"
	@echo "  uninstall-user Uninstall for current user"
	@echo "  dev           Build and run in development mode"
	@echo "  test          Run tests"
	@echo "  deps          Update Go dependencies"
	@echo "  help          Show this help message"

.PHONY: all build clean install uninstall install-user uninstall-user dev test deps help