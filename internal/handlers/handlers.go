package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"dconn.dev/internal/config"
	"dconn.dev/internal/middleware"
	"dconn.dev/internal/services"
)

// SetupRoutes configures all routes and returns the router
func SetupRoutes(cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Recovery)
	r.Use(middleware.Logger)

	// Initialize services
	mapService := services.NewMapService(cfg.GameMap)
	gameService := services.NewGameService(mapService)
	projectService := services.NewProjectService(cfg.Projects)

	// Initialize world service for chunk-based maps
	worldService, err := services.NewWorldService(cfg.DataPath)
	if err != nil {
		log.Printf("Warning: Failed to initialize WorldService: %v", err)
	}

	// Initialize handlers
	gameHandler := NewGameHandler(gameService, mapService)
	projectHandler := NewProjectHandler(projectService)
	var worldHandler *WorldHandler
	if worldService != nil {
		worldHandler = NewWorldHandler(worldService)
	}

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Game endpoints (legacy)
		r.Get("/game/init", gameHandler.InitGame)
		r.Post("/game/move", gameHandler.Move)
		r.Get("/game/map", gameHandler.GetFullMap)

		// World/chunk endpoints (new)
		if worldHandler != nil {
			r.Get("/world", worldHandler.GetWorld)
			r.Get("/chunks/{x}/{y}", worldHandler.GetChunk)
		}

		// Project endpoints
		r.Get("/projects", projectHandler.ListProjects)
		r.Get("/projects/{id}", projectHandler.GetProject)

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
	})

	// Static files
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// Serve index.html at root
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("static", "index.html"))
	})

	return r
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

// respondError writes an error JSON response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
