package models

// World represents the entire game world manifest
type World struct {
	ChunkSize       int                 `json:"chunk_size"`
	SpawnChunk      [2]int              `json:"spawn_chunk"`
	SpawnLocal      [2]int              `json:"spawn_local"`
	TileDefinitions map[string]Tile     `json:"tile_definitions"`
	Chunks          map[string]ChunkRef `json:"chunks"`
}

// ChunkRef is a reference to a chunk file in the manifest
type ChunkRef struct {
	Name string `json:"name"`
	File string `json:"file"`
}

// Chunk represents a single map chunk
type Chunk struct {
	Tiles [][]string `json:"tiles"`
	Zones []Zone     `json:"zones"`
}

// ChunkResponse is what we send to the client
type ChunkResponse struct {
	X     int        `json:"x"`
	Y     int        `json:"y"`
	Tiles [][]string `json:"tiles"`
	Zones []Zone     `json:"zones"`
}

// WorldResponse is the manifest sent to the client
type WorldResponse struct {
	ChunkSize       int               `json:"chunk_size"`
	SpawnChunk      [2]int            `json:"spawn_chunk"`
	SpawnLocal      [2]int            `json:"spawn_local"`
	TileDefinitions map[string]Tile   `json:"tile_definitions"`
	AvailableChunks map[string]string `json:"available_chunks"` // "x,y" -> name
}

// GameMap represents the entire game world (legacy, kept for compatibility)
type GameMap struct {
	Width           int                `json:"width"`
	Height          int                `json:"height"`
	Tiles           [][]string         `json:"tiles"`
	TileDefinitions map[string]Tile    `json:"tile_definitions"`
	Zones           []Zone             `json:"zones"`
	Spawn           Position           `json:"spawn"`
}

// Zone represents an interactive area on the map
type Zone struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Bounds      Bounds `json:"bounds"`
	ProjectID   string `json:"project_id,omitempty"`
}

// Bounds defines a rectangular area
type Bounds struct {
	MinX int `json:"min_x"`
	MaxX int `json:"max_x"`
	MinY int `json:"min_y"`
	MaxY int `json:"max_y"`
}

// ViewportData represents the visible area around the player
type ViewportData struct {
	Tiles       [][]RenderedTile `json:"tiles"`
	PlayerX     int              `json:"player_x"` // Relative to viewport
	PlayerY     int              `json:"player_y"` // Relative to viewport
	CurrentZone *Zone            `json:"current_zone,omitempty"`
}
