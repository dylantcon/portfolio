package generation

// Point represents a 2D coordinate
type Point struct {
	X, Y int
}

// Add returns a new point offset by dx, dy
func (p Point) Add(dx, dy int) Point {
	return Point{p.X + dx, p.Y + dy}
}

// Adjacent returns the 4 cardinal neighbors
func (p Point) Adjacent() []Point {
	return []Point{
		{p.X, p.Y - 1}, // N
		{p.X + 1, p.Y}, // E
		{p.X, p.Y + 1}, // S
		{p.X - 1, p.Y}, // W
	}
}

// Direction represents cardinal directions
type Direction int

const (
	North Direction = iota
	East
	South
	West
)

// Opposite returns the opposite direction
func (d Direction) Opposite() Direction {
	return (d + 2) % 4
}

// Delta returns the x,y offset for moving in this direction
func (d Direction) Delta() (int, int) {
	switch d {
	case North:
		return 0, -1
	case East:
		return 1, 0
	case South:
		return 0, 1
	case West:
		return -1, 0
	}
	return 0, 0
}

// Bounds represents a rectangular region
type Bounds struct {
	MinX, MinY, MaxX, MaxY int
}

// Width returns the width of the bounds
func (b Bounds) Width() int {
	return b.MaxX - b.MinX + 1
}

// Height returns the height of the bounds
func (b Bounds) Height() int {
	return b.MaxY - b.MinY + 1
}

// Contains checks if a point is within bounds
func (b Bounds) Contains(p Point) bool {
	return p.X >= b.MinX && p.X <= b.MaxX && p.Y >= b.MinY && p.Y <= b.MaxY
}

// Overlaps checks if two bounds intersect
func (b Bounds) Overlaps(other Bounds) bool {
	return b.MinX <= other.MaxX && b.MaxX >= other.MinX &&
		b.MinY <= other.MaxY && b.MaxY >= other.MinY
}

// Expand returns bounds grown by n tiles in each direction
func (b Bounds) Expand(n int) Bounds {
	return Bounds{b.MinX - n, b.MinY - n, b.MaxX + n, b.MaxY + n}
}

// Center returns the center point of the bounds
func (b Bounds) Center() Point {
	return Point{(b.MinX + b.MaxX) / 2, (b.MinY + b.MaxY) / 2}
}

// Grid represents a 2D tile grid that components render onto
type Grid struct {
	Width, Height int
	Tiles         [][]string
	Walkable      [][]bool // Cached walkability for pathfinding
}

// NewGrid creates a new grid filled with a default tile
func NewGrid(width, height int, defaultTile string, walkable bool) *Grid {
	tiles := make([][]string, height)
	walk := make([][]bool, height)
	for y := 0; y < height; y++ {
		tiles[y] = make([]string, width)
		walk[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			tiles[y][x] = defaultTile
			walk[y][x] = walkable
		}
	}
	return &Grid{Width: width, Height: height, Tiles: tiles, Walkable: walk}
}

// InBounds checks if a point is within the grid
func (g *Grid) InBounds(p Point) bool {
	return p.X >= 0 && p.X < g.Width && p.Y >= 0 && p.Y < g.Height
}

// Set sets a tile at a position
func (g *Grid) Set(p Point, tile string, walkable bool) {
	if g.InBounds(p) {
		g.Tiles[p.Y][p.X] = tile
		g.Walkable[p.Y][p.X] = walkable
	}
}

// Get returns the tile at a position
func (g *Grid) Get(p Point) string {
	if g.InBounds(p) {
		return g.Tiles[p.Y][p.X]
	}
	return ""
}

// IsWalkable checks if a position is walkable
func (g *Grid) IsWalkable(p Point) bool {
	if g.InBounds(p) {
		return g.Walkable[p.Y][p.X]
	}
	return false
}

// Anchor represents a connection point on a component
type Anchor struct {
	Position Point
	Direction Direction // Which direction the anchor faces (for path connections)
}

// Zone represents an interactive area tied to a project
type Zone struct {
	Name        string
	Description string
	Bounds      Bounds
	ProjectID   string
}
