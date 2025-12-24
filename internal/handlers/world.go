package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"dconn.dev/internal/services"
)

// WorldHandler handles world and chunk endpoints
type WorldHandler struct {
	worldService *services.WorldService
}

// NewWorldHandler creates a new WorldHandler
func NewWorldHandler(ws *services.WorldService) *WorldHandler {
	return &WorldHandler{worldService: ws}
}

// GetWorld handles GET /api/world - returns world manifest
func (h *WorldHandler) GetWorld(w http.ResponseWriter, r *http.Request) {
	world := h.worldService.GetWorldResponse()
	respondJSON(w, http.StatusOK, world)
}

// GetChunk handles GET /api/chunks/{x}/{y} - returns a specific chunk
func (h *WorldHandler) GetChunk(w http.ResponseWriter, r *http.Request) {
	xStr := chi.URLParam(r, "x")
	yStr := chi.URLParam(r, "y")

	x, err := strconv.Atoi(xStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid x coordinate")
		return
	}

	y, err := strconv.Atoi(yStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid y coordinate")
		return
	}

	chunk, err := h.worldService.GetChunk(x, y)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, chunk)
}
