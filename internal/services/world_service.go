package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dconn.dev/internal/models"
)

// WorldService manages the chunk-based world
type WorldService struct {
	world     *models.World
	dataPath  string
	chunks    map[string]*models.Chunk // cached chunks
	signposts []models.Signpost        // seam markers with directional hints
}

// NewWorldService creates a new WorldService
func NewWorldService(dataPath string) (*WorldService, error) {
	ws := &WorldService{
		dataPath: dataPath,
		chunks:   make(map[string]*models.Chunk),
	}

	if err := ws.loadWorld(); err != nil {
		return nil, err
	}

	ws.loadSignposts()

	return ws, nil
}

// loadWorld loads the world manifest
func (ws *WorldService) loadWorld() error {
	path := filepath.Join(ws.dataPath, "world.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read world.json: %w", err)
	}

	ws.world = &models.World{}
	if err := json.Unmarshal(data, ws.world); err != nil {
		return fmt.Errorf("failed to parse world.json: %w", err)
	}

	return nil
}

// loadSignposts loads the seam-marker registry. A missing file is non-fatal:
// the server still serves the world with an empty signpost list (e.g. before
// the first `make generate`).
func (ws *WorldService) loadSignposts() {
	path := filepath.Join(ws.dataPath, "signposts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var file models.SignpostFile
	if err := json.Unmarshal(data, &file); err != nil {
		return
	}
	ws.signposts = file.Signposts
}

// GetWorldResponse returns the world manifest for the client
func (ws *WorldService) GetWorldResponse() *models.WorldResponse {
	available := make(map[string]string)
	for key, ref := range ws.world.Chunks {
		available[key] = ref.Name
	}

	return &models.WorldResponse{
		ChunkSize:       ws.world.ChunkSize,
		SpawnChunk:      ws.world.SpawnChunk,
		SpawnLocal:      ws.world.SpawnLocal,
		TileDefinitions: ws.world.TileDefinitions,
		AvailableChunks: available,
		Signposts:       ws.signposts,
	}
}

// GetChunk returns a chunk by grid coordinates
func (ws *WorldService) GetChunk(x, y int) (*models.ChunkResponse, error) {
	key := fmt.Sprintf("%d,%d", x, y)

	// Check if chunk exists in manifest
	ref, exists := ws.world.Chunks[key]
	if !exists {
		return nil, fmt.Errorf("chunk %s not found", key)
	}

	// Check cache
	if chunk, cached := ws.chunks[key]; cached {
		return &models.ChunkResponse{
			X:     x,
			Y:     y,
			Tiles: chunk.Tiles,
			Zones: chunk.Zones,
		}, nil
	}

	// Load from file
	path := filepath.Join(ws.dataPath, ref.File)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk file: %w", err)
	}

	chunk := &models.Chunk{}
	if err := json.Unmarshal(data, chunk); err != nil {
		return nil, fmt.Errorf("failed to parse chunk file: %w", err)
	}

	// Cache it
	ws.chunks[key] = chunk

	return &models.ChunkResponse{
		X:     x,
		Y:     y,
		Tiles: chunk.Tiles,
		Zones: chunk.Zones,
	}, nil
}

// ChunkExists checks if a chunk exists at the given coordinates
func (ws *WorldService) ChunkExists(x, y int) bool {
	key := fmt.Sprintf("%d,%d", x, y)
	_, exists := ws.world.Chunks[key]
	return exists
}

// GetTileDefinitions returns the global tile definitions
func (ws *WorldService) GetTileDefinitions() map[string]models.Tile {
	return ws.world.TileDefinitions
}
