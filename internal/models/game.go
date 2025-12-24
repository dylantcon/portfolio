package models

// Position represents a coordinate on the game map
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// GameState represents the current state of the game
type GameState struct {
	PlayerPosition Position `json:"player_position"`
	ViewportSize   int      `json:"viewport_size"`
	CurrentZone    *Zone    `json:"current_zone,omitempty"`
}

// Tile represents a single tile on the map
type Tile struct {
	Character string `json:"char"`
	Color     string `json:"color"`
	Type      string `json:"type"` // water, grass, sand, building, etc.
	Walkable  bool   `json:"walkable"`
}

// RenderedTile represents a tile as sent to the client
type RenderedTile struct {
	Character string `json:"char"`
	Color     string `json:"color"`
}
