.PHONY: build run clean restart

# Build the server binary to bin/server
build:
	go build -o bin/server ./cmd/server

# Run the server (for development)
run: build
	./bin/server

# Clean build artifacts
clean:
	rm -f bin/server

# Build and restart (assumes systemd or similar - adjust as needed)
restart: build
	@echo "Binary built. Restart your server process to apply changes."
	@echo "Example: kill -HUP \$$(pgrep -f 'bin/server') or restart your service"
