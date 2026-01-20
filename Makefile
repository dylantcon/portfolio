.PHONY: build run clean restart generate

# Build the server binary to bin/server
build:
	go build -o bin/server ./cmd/server

# Regenerate map chunk JSON files
generate:
	go build -o bin/generate ./cmd/generate
	./bin/generate ./data

# Run the server (for development)
run: build
	./bin/server

# Clean build artifacts
clean:
	rm -f bin/server bin/generate

# Build and restart (assumes systemd or similar - adjust as needed)
restart: build
	@echo "Binary built. Restart your server process to apply changes."
	@echo "Example: kill -HUP \$$(pgrep -f 'bin/server') or restart your service"
