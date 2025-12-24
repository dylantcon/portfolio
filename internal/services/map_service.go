package services

import (
	"dconn.dev/internal/models"
)

// MapService handles map-related operations
type MapService struct {
	gameMap *models.GameMap
}

// NewMapService creates a new MapService
func NewMapService(gm *models.GameMap) *MapService {
	return &MapService{gameMap: gm}
}

// GetSpawnPoint returns the spawn position
func (s *MapService) GetSpawnPoint() models.Position {
	return s.gameMap.Spawn
}

// GetViewport returns the visible tiles around a center position
// width and height specify the viewport dimensions
func (s *MapService) GetViewport(center models.Position, width, height int) *models.ViewportData {
	halfWidth := width / 2
	halfHeight := height / 2
	viewport := &models.ViewportData{
		Tiles:   make([][]models.RenderedTile, height),
		PlayerX: halfWidth,
		PlayerY: halfHeight,
	}

	for y := 0; y < height; y++ {
		viewport.Tiles[y] = make([]models.RenderedTile, width)
		for x := 0; x < width; x++ {
			mapX := center.X - halfWidth + x
			mapY := center.Y - halfHeight + y

			tile := s.getTileAt(mapX, mapY)
			viewport.Tiles[y][x] = models.RenderedTile{
				Character: tile.Character,
				Color:     tile.Color,
			}
		}
	}

	// Check for zone at player position
	viewport.CurrentZone = s.GetZoneAt(center)

	return viewport
}

// getTileAt returns the tile at a specific position
func (s *MapService) getTileAt(x, y int) models.Tile {
	// Return gray ? for out-of-bounds areas
	if x < 0 || y < 0 || x >= s.gameMap.Width || y >= s.gameMap.Height {
		return models.Tile{
			Character: "?",
			Color:     "#2a2a2a",
			Type:      "void",
			Walkable:  false,
		}
	}

	// Get the character at this position
	char := s.gameMap.Tiles[y][x]

	// Look up the tile definition
	if tileDef, exists := s.gameMap.TileDefinitions[char]; exists {
		return tileDef
	}

	// Default tile if no definition found
	return models.Tile{
		Character: char,
		Color:     "#808080",
		Type:      "unknown",
		Walkable:  true,
	}
}

// IsWalkable checks if a position can be walked on
func (s *MapService) IsWalkable(pos models.Position) bool {
	if pos.X < 0 || pos.Y < 0 || pos.X >= s.gameMap.Width || pos.Y >= s.gameMap.Height {
		return false
	}

	tile := s.getTileAt(pos.X, pos.Y)
	return tile.Walkable
}

// GetZoneAt returns the zone at a specific position, or nil if none
func (s *MapService) GetZoneAt(pos models.Position) *models.Zone {
	for i := range s.gameMap.Zones {
		zone := &s.gameMap.Zones[i]
		if pos.X >= zone.Bounds.MinX && pos.X <= zone.Bounds.MaxX &&
			pos.Y >= zone.Bounds.MinY && pos.Y <= zone.Bounds.MaxY {
			return zone
		}
	}
	return nil
}

// GetAllZones returns all zones on the map
func (s *MapService) GetAllZones() []models.Zone {
	return s.gameMap.Zones
}

// FullMapData is the complete map data for client-side rendering
type FullMapData struct {
	Width           int                    `json:"width"`
	Height          int                    `json:"height"`
	Spawn           models.Position        `json:"spawn"`
	Tiles           [][]models.RenderedTile `json:"tiles"`
	TileDefinitions map[string]models.Tile `json:"tile_definitions"`
	Zones           []models.Zone          `json:"zones"`
}

// GetFullMapData returns the entire map for client-side caching
func (s *MapService) GetFullMapData() *FullMapData {
	// Pre-render all tiles
	tiles := make([][]models.RenderedTile, s.gameMap.Height)
	for y := 0; y < s.gameMap.Height; y++ {
		tiles[y] = make([]models.RenderedTile, s.gameMap.Width)
		for x := 0; x < s.gameMap.Width; x++ {
			tile := s.getTileAt(x, y)
			tiles[y][x] = models.RenderedTile{
				Character: tile.Character,
				Color:     tile.Color,
			}
		}
	}

	return &FullMapData{
		Width:           s.gameMap.Width,
		Height:          s.gameMap.Height,
		Spawn:           s.gameMap.Spawn,
		Tiles:           tiles,
		TileDefinitions: s.gameMap.TileDefinitions,
		Zones:           s.gameMap.Zones,
	}
}
