package config

import (
	"encoding/json"
	"os"

	"dconn.dev/internal/models"
)

// Config holds all application configuration
type Config struct {
	ServerAddr string
	DataPath   string
	GameMap    *models.GameMap
	Projects   *models.ProjectList
	GameConfig *GameConfig
	Spotify    SpotifyConfig
}

// SpotifyConfig holds the credentials for the "now playing" badge. All three are
// read from the environment (or .env). If any is empty, the feature is disabled.
type SpotifyConfig struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
}

// Enabled reports whether all Spotify credentials are present.
func (s SpotifyConfig) Enabled() bool {
	return s.ClientID != "" && s.ClientSecret != "" && s.RefreshToken != ""
}

// GameConfig holds game-specific settings
type GameConfig struct {
	ViewportSize int    `json:"viewport_size"`
	PlayerChar   string `json:"player_char"`
	PlayerColor  string `json:"player_color"`
	MoveDelayMs  int    `json:"move_delay_ms"`
	Theme        Theme  `json:"theme"`
}

// Theme holds color scheme settings
type Theme struct {
	Background string `json:"background"`
	Text       string `json:"text"`
	Accent     string `json:"accent"`
	Error      string `json:"error"`
}

// Load reads and parses all configuration files
func Load() *Config {
	// Load .env (if present) before reading any environment variables.
	LoadDotEnv(".env")

	gameMap := loadGameMap()
	projects := loadProjects()
	gameConfig := loadGameConfig()

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = ":8080"
	}

	return &Config{
		ServerAddr: serverAddr,
		DataPath:   "data",
		GameMap:    gameMap,
		Projects:   projects,
		GameConfig: gameConfig,
		Spotify: SpotifyConfig{
			ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
			ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
			RefreshToken: os.Getenv("SPOTIFY_REFRESH_TOKEN"),
		},
	}
}

// loadGameMap reads the map.json file
func loadGameMap() *models.GameMap {
	data, err := os.ReadFile("data/map.json")
	if err != nil {
		panic("Failed to load map.json: " + err.Error())
	}

	var gameMap models.GameMap
	if err := json.Unmarshal(data, &gameMap); err != nil {
		panic("Failed to parse map.json: " + err.Error())
	}

	return &gameMap
}

// loadProjects reads the projects.json file
func loadProjects() *models.ProjectList {
	data, err := os.ReadFile("data/projects.json")
	if err != nil {
		panic("Failed to load projects.json: " + err.Error())
	}

	var projects models.ProjectList
	if err := json.Unmarshal(data, &projects); err != nil {
		panic("Failed to parse projects.json: " + err.Error())
	}

	return &projects
}

// loadGameConfig reads the config.json file
func loadGameConfig() *GameConfig {
	data, err := os.ReadFile("data/config.json")
	if err != nil {
		panic("Failed to load config.json: " + err.Error())
	}

	var gameConfig GameConfig
	if err := json.Unmarshal(data, &gameConfig); err != nil {
		panic("Failed to parse config.json: " + err.Error())
	}

	return &gameConfig
}
