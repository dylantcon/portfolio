.PHONY: build run clean restart restart-service generate spotify-auth

# systemd unit for the running server. Override on the CLI if it differs,
# e.g. `make generate SERVICE=my-unit.service`.
SERVICE ?= dconn-dev.service

# Build the server binary to bin/server
build:
	go build -o bin/server ./cmd/server

# Regenerate map chunk JSON files, then restart the service so the running
# server picks up the new data (projects and chunks are loaded/cached in memory
# at startup, so a restart is required for regenerated data to take effect).
generate:
	go build -o bin/generate ./cmd/generate
	./bin/generate ./data
	@$(MAKE) --no-print-directory restart-service

# Restart the service if it is installed and active, so regenerated data is
# reloaded. No-ops with a hint when the unit isn't running (e.g. local dev where
# the server is started via `make run`).
restart-service:
	@if systemctl is-active --quiet $(SERVICE); then \
		echo "Restarting $(SERVICE) to reload regenerated data..."; \
		sudo systemctl restart $(SERVICE) && echo "Restarted $(SERVICE)."; \
	else \
		echo "$(SERVICE) is not active; skipping restart (start the server manually to see changes)."; \
	fi

# Run the server (for development)
run: build
	./bin/server

# One-time: obtain a Spotify refresh token for the "now playing" badge.
# Requires SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET set in .env first.
spotify-auth:
	go run ./cmd/spotify-auth

# Clean build artifacts
clean:
	rm -f bin/server bin/generate

# Build and restart (assumes systemd or similar - adjust as needed)
restart: build
	@echo "Binary built. Restart your server process to apply changes."
	@echo "Example: kill -HUP \$$(pgrep -f 'bin/server') or restart your service"
