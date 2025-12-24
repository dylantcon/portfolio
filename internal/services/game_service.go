package services

import (
	"fmt"

	"dconn.dev/internal/models"
)

// GameService handles game logic
type GameService struct {
	mapService *MapService
}

// NewGameService creates a new GameService
func NewGameService(ms *MapService) *GameService {
	return &GameService{mapService: ms}
}

// NewGame initializes a new game state at the spawn point
func (s *GameService) NewGame() *models.GameState {
	spawn := s.mapService.GetSpawnPoint()
	return &models.GameState{
		PlayerPosition: spawn,
		ViewportSize:   15,
	}
}

// Move attempts to move the player in a direction
// Returns the new position and any error
func (s *GameService) Move(pos models.Position, direction string) (models.Position, error) {
	newPos := pos

	switch direction {
	case "north", "w", "W":
		newPos.Y--
	case "south", "s", "S":
		newPos.Y++
	case "east", "d", "D":
		newPos.X++
	case "west", "a", "A":
		newPos.X--
	default:
		return pos, fmt.Errorf("invalid direction: %s", direction)
	}

	if !s.mapService.IsWalkable(newPos) {
		return pos, fmt.Errorf("cannot walk there")
	}

	return newPos, nil
}
