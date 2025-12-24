package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"dconn.dev/internal/models"
	"dconn.dev/internal/services"
)

// GameHandler handles game-related endpoints
type GameHandler struct {
	gameService *services.GameService
	mapService  *services.MapService
}

// NewGameHandler creates a new GameHandler
func NewGameHandler(gs *services.GameService, ms *services.MapService) *GameHandler {
	return &GameHandler{
		gameService: gs,
		mapService:  ms,
	}
}

// InitGame handles GET /api/game/init
func (h *GameHandler) InitGame(w http.ResponseWriter, r *http.Request) {
	// Parse viewport dimensions from query params
	width := parseIntParam(r, "width", 40)
	height := parseIntParam(r, "height", 20)

	// Clamp to reasonable values
	width = clamp(width, 10, 200)
	height = clamp(height, 10, 100)

	state := h.gameService.NewGame()
	viewport := h.mapService.GetViewport(state.PlayerPosition, width, height)

	respondJSON(w, http.StatusOK, viewport)
}

// Move handles POST /api/game/move
func (h *GameHandler) Move(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Direction string          `json:"direction"`
		Position  models.Position `json:"position"`
		Width     int             `json:"width"`
		Height    int             `json:"height"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Default dimensions if not provided
	if req.Width == 0 {
		req.Width = 40
	}
	if req.Height == 0 {
		req.Height = 20
	}

	// Clamp to reasonable values
	req.Width = clamp(req.Width, 10, 200)
	req.Height = clamp(req.Height, 10, 100)

	newPos, err := h.gameService.Move(req.Position, req.Direction)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	viewport := h.mapService.GetViewport(newPos, req.Width, req.Height)
	respondJSON(w, http.StatusOK, viewport)
}

// GetFullMap handles GET /api/game/map - returns full map for client-side rendering
func (h *GameHandler) GetFullMap(w http.ResponseWriter, r *http.Request) {
	mapData := h.mapService.GetFullMapData()
	respondJSON(w, http.StatusOK, mapData)
}

// parseIntParam parses an integer query parameter with a default value
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

// clamp limits a value to a range
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
