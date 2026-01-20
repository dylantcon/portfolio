package generation

// Palette defines the tiles available for a biome
type Palette struct {
	// Terrain
	Grass     string
	Sand      string
	Water     string
	DeepWater string
	Snow      string
	Mountain  string
	Peak      string

	// Vegetation
	Tree     string
	PineTree string
	Bush     string

	// Structures
	Building      string
	WhiteBuilding string
	WoodWall      string
	Door          string
	Pillar        string

	// Infrastructure
	Path        string
	Cobblestone string
	Dock        string
	Bridge      string

	// Special
	Star   string
	Marker string
	Empty  string

	// Additional details
	Window    string
	WoodFloor string
	Chimney   string
}

// DefaultPalette returns the standard tile palette
func DefaultPalette() *Palette {
	return &Palette{
		Grass:         "^",
		Sand:          ".",
		Water:         "~",
		DeepWater:     "≈",
		Snow:          "s",
		Mountain:      "M",
		Peak:          "A",
		Tree:          "T",
		PineTree:      "t",
		Bush:          ";",
		Building:      "#",
		WhiteBuilding: "B",
		WoodWall:      "W",
		Door:          "D",
		Pillar:        "|",
		Path:          "+",
		Cobblestone:   "o",
		Dock:          "=",
		Bridge:        "n",
		Star:          "*",
		Marker:        "@",
		Empty:         " ",
		Window:        "%",
		WoodFloor:     "░",
		Chimney:       "H",
	}
}

// BiomeType identifies the type of terrain
type BiomeType string

const (
	BiomeGrassland BiomeType = "grassland"
	BiomeMountain  BiomeType = "mountain"
	BiomeCoastal   BiomeType = "coastal"
	BiomeForest    BiomeType = "forest"
	BiomeUrban     BiomeType = "urban"
	BiomeCastle    BiomeType = "castle"
)

// Biome defines generation rules for a terrain type
type Biome struct {
	Type BiomeType

	// Base terrain
	BaseTile     string
	BaseWalkable bool

	// Allowed components
	AllowedStructures []string // "building", "cabin", "tower", "courtyard", "shrine"
	AllowedTerrain    []string // "grove", "clearing", "lake", "mountain_range", "shoreline"
	AllowedInfra      []string // "plaza", "dock", "bridge"

	// Decoration settings
	TreeType    string
	TreeDensity float64
	BushDensity float64

	// Edge behavior - which edges have water/mountains/etc
	Shorelines []Direction // Edges that have water
	Mountains  []Direction // Edges that have mountains
}

// GetBiome returns the biome configuration for a type
func GetBiome(t BiomeType) *Biome {
	switch t {
	case BiomeGrassland:
		return &Biome{
			Type:              BiomeGrassland,
			BaseTile:          "^",
			BaseWalkable:      true,
			AllowedStructures: []string{"building", "cabin", "shrine"},
			AllowedTerrain:    []string{"grove", "clearing"},
			AllowedInfra:      []string{"plaza", "bridge"},
			TreeType:          "T",
			TreeDensity:       0.03,
			BushDensity:       0.01,
		}

	case BiomeMountain:
		return &Biome{
			Type:              BiomeMountain,
			BaseTile:          "^",
			BaseWalkable:      true,
			AllowedStructures: []string{"cabin", "tower", "shrine"},
			AllowedTerrain:    []string{"mountain_range", "clearing"},
			AllowedInfra:      []string{"bridge"},
			TreeType:          "t",
			TreeDensity:       0.05,
			BushDensity:       0.0,
		}

	case BiomeCoastal:
		return &Biome{
			Type:              BiomeCoastal,
			BaseTile:          "^",
			BaseWalkable:      true,
			AllowedStructures: []string{"building", "cabin"},
			AllowedTerrain:    []string{"shoreline", "clearing"},
			AllowedInfra:      []string{"plaza", "dock", "bridge"},
			TreeType:          "T",
			TreeDensity:       0.02,
			BushDensity:       0.02,
		}

	case BiomeForest:
		return &Biome{
			Type:              BiomeForest,
			BaseTile:          "^",
			BaseWalkable:      true,
			AllowedStructures: []string{"cabin", "shrine"},
			AllowedTerrain:    []string{"grove", "clearing"},
			AllowedInfra:      []string{"bridge"},
			TreeType:          "T",
			TreeDensity:       0.15,
			BushDensity:       0.05,
		}

	case BiomeUrban:
		return &Biome{
			Type:              BiomeUrban,
			BaseTile:          "^",
			BaseWalkable:      true,
			AllowedStructures: []string{"building", "tower", "courtyard"},
			AllowedTerrain:    []string{"clearing"},
			AllowedInfra:      []string{"plaza"},
			TreeType:          "T",
			TreeDensity:       0.01,
			BushDensity:       0.02,
		}

	case BiomeCastle:
		return &Biome{
			Type:              BiomeCastle,
			BaseTile:          "^",
			BaseWalkable:      true,
			AllowedStructures: []string{"building", "tower", "courtyard", "shrine"},
			AllowedTerrain:    []string{"clearing"},
			AllowedInfra:      []string{"plaza", "bridge"},
			TreeType:          "T",
			TreeDensity:       0.02,
			BushDensity:       0.01,
		}

	default:
		return GetBiome(BiomeGrassland)
	}
}

// ChunkConfig defines what should be generated for a chunk
type ChunkConfig struct {
	// Identity
	ChunkX, ChunkY int
	Seed           uint64

	// Terrain
	Biome      BiomeType
	Shorelines []Direction // Which edges have water

	// Connectivity - which edges connect to other chunks
	Connections   []Direction
	SignpostHints map[Direction]string // Hints for signposts at each exit

	// Projects to place in this chunk
	Projects []ProjectPlacement
}

// ProjectPlacement defines where a project should be placed
type ProjectPlacement struct {
	ProjectID   string
	Name        string
	Description string
	Structure   string // "building", "tower", "shrine", "courtyard"
	Size        int    // Relative size (1-3)
}

// ChunkDefinition is the output - matches the JSON format
type ChunkDefinition struct {
	Tiles [][]string `json:"tiles"`
	Zones []ZoneDef  `json:"zones"`
}

// ZoneDef matches the JSON zone format
type ZoneDef struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Bounds      BoundsDef `json:"bounds"`
	ProjectID   string    `json:"project_id,omitempty"`
}

// BoundsDef matches the JSON bounds format
type BoundsDef struct {
	MinX int `json:"min_x"`
	MaxX int `json:"max_x"`
	MinY int `json:"min_y"`
	MaxY int `json:"max_y"`
}
