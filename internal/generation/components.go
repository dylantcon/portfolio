package generation

// Component is the interface all placeable components implement
type Component interface {
	// Render draws the component onto the grid
	Render(g *Grid, palette *Palette)
	// GetBounds returns the bounding box of this component
	GetBounds() Bounds
	// GetAnchors returns connection points for paths
	GetAnchors() []Anchor
	// GetZone returns the zone if this component has a project, nil otherwise
	GetZone() *Zone
}

// ---- Terrain Components ----

// Shoreline creates a water->sand->grass gradient along a chunk edge
type Shoreline struct {
	Side       Direction
	WaterDepth int // How many tiles of water
	SandDepth  int // How many tiles of sand
	bounds     Bounds
}

func NewShoreline(side Direction, waterDepth, sandDepth int, chunkSize int) *Shoreline {
	s := &Shoreline{Side: side, WaterDepth: waterDepth, SandDepth: sandDepth}

	switch side {
	case North:
		s.bounds = Bounds{0, 0, chunkSize - 1, waterDepth + sandDepth - 1}
	case South:
		s.bounds = Bounds{0, chunkSize - waterDepth - sandDepth, chunkSize - 1, chunkSize - 1}
	case East:
		s.bounds = Bounds{chunkSize - waterDepth - sandDepth, 0, chunkSize - 1, chunkSize - 1}
	case West:
		s.bounds = Bounds{0, 0, waterDepth + sandDepth - 1, chunkSize - 1}
	}
	return s
}

func (s *Shoreline) Render(g *Grid, p *Palette) {
	for y := s.bounds.MinY; y <= s.bounds.MaxY; y++ {
		for x := s.bounds.MinX; x <= s.bounds.MaxX; x++ {
			var depth int
			switch s.Side {
			case North:
				depth = y - s.bounds.MinY
			case South:
				depth = s.bounds.MaxY - y
			case East:
				depth = s.bounds.MaxX - x
			case West:
				depth = x - s.bounds.MinX
			}

			if depth < s.WaterDepth {
				if depth < s.WaterDepth/2 {
					g.Set(Point{x, y}, p.DeepWater, false)
				} else {
					g.Set(Point{x, y}, p.Water, false)
				}
			} else {
				g.Set(Point{x, y}, p.Sand, true)
			}
		}
	}
}

func (s *Shoreline) GetBounds() Bounds   { return s.bounds }
func (s *Shoreline) GetAnchors() []Anchor { return nil }
func (s *Shoreline) GetZone() *Zone       { return nil }

// MountainRange creates impassable mountains with defined passes
type MountainRange struct {
	bounds      Bounds
	passes      []Point // Locations where paths can go through
	snowLine    int     // Y offset where snow starts
}

func NewMountainRange(bounds Bounds, passes []Point, snowLine int) *MountainRange {
	return &MountainRange{bounds: bounds, passes: passes, snowLine: snowLine}
}

func (m *MountainRange) Render(g *Grid, p *Palette) {
	passSet := make(map[Point]bool)
	for _, pass := range m.passes {
		passSet[pass] = true
		// Make pass area slightly wider
		for _, adj := range pass.Adjacent() {
			passSet[adj] = true
		}
	}

	for y := m.bounds.MinY; y <= m.bounds.MaxY; y++ {
		for x := m.bounds.MinX; x <= m.bounds.MaxX; x++ {
			pt := Point{x, y}
			if passSet[pt] {
				g.Set(pt, p.Path, true)
				continue
			}

			distFromTop := y - m.bounds.MinY
			if distFromTop < m.snowLine {
				g.Set(pt, p.Snow, false)
			} else if distFromTop < m.snowLine+2 {
				g.Set(pt, p.Peak, false)
			} else {
				g.Set(pt, p.Mountain, false)
			}
		}
	}
}

func (m *MountainRange) GetBounds() Bounds { return m.bounds }
func (m *MountainRange) GetAnchors() []Anchor {
	anchors := make([]Anchor, len(m.passes))
	for i, pass := range m.passes {
		anchors[i] = Anchor{Position: pass, Direction: South}
	}
	return anchors
}
func (m *MountainRange) GetZone() *Zone { return nil }

// Grove creates a cluster of trees
type Grove struct {
	bounds   Bounds
	density  float64
	treeType string // Which tree tile to use
	rng      *RNG
}

func NewGrove(bounds Bounds, density float64, treeType string, rng *RNG) *Grove {
	return &Grove{bounds: bounds, density: density, treeType: treeType, rng: rng}
}

func (gr *Grove) Render(g *Grid, p *Palette) {
	tree := gr.treeType
	if tree == "" {
		tree = p.Tree
	}
	g.ScatterOnTile(gr.bounds, p.Grass, tree, false, gr.density, gr.rng)
}

func (gr *Grove) GetBounds() Bounds    { return gr.bounds }
func (gr *Grove) GetAnchors() []Anchor { return nil }
func (gr *Grove) GetZone() *Zone       { return nil }

// Clearing creates an open space (ensures an area is walkable grass)
type Clearing struct {
	center Point
	radius int
}

func NewClearing(center Point, radius int) *Clearing {
	return &Clearing{center: center, radius: radius}
}

func (c *Clearing) Render(g *Grid, p *Palette) {
	for dy := -c.radius; dy <= c.radius; dy++ {
		for dx := -c.radius; dx <= c.radius; dx++ {
			if dx*dx+dy*dy <= c.radius*c.radius {
				g.Set(Point{c.center.X + dx, c.center.Y + dy}, p.Grass, true)
			}
		}
	}
}

func (c *Clearing) GetBounds() Bounds {
	return Bounds{c.center.X - c.radius, c.center.Y - c.radius,
		c.center.X + c.radius, c.center.Y + c.radius}
}
func (c *Clearing) GetAnchors() []Anchor {
	return []Anchor{{Position: c.center, Direction: South}}
}
func (c *Clearing) GetZone() *Zone { return nil }

// ---- Structure Components ----

// Building creates a rectangular structure with walls and a door
type Building struct {
	bounds      Bounds
	style       string // "stone", "white", "wood"
	entranceDir Direction
	zone        *Zone
}

func NewBuilding(bounds Bounds, style string, entranceDir Direction, zone *Zone) *Building {
	return &Building{bounds: bounds, style: style, entranceDir: entranceDir, zone: zone}
}

func (b *Building) Render(g *Grid, p *Palette) {
	// Determine wall tile based on style
	wallTile := p.Building
	switch b.style {
	case "white":
		wallTile = p.WhiteBuilding
	case "wood":
		wallTile = p.WoodWall
	}

	// Fill interior with cobblestone floor
	interior := Bounds{b.bounds.MinX + 1, b.bounds.MinY + 1, b.bounds.MaxX - 1, b.bounds.MaxY - 1}
	g.Rect(interior, p.Cobblestone, true)

	// Draw walls
	g.RectOutline(b.bounds, wallTile, false)

	// Add windows on walls (not on corners or door side)
	doorPos := b.getDoorPosition()
	width := b.bounds.MaxX - b.bounds.MinX
	height := b.bounds.MaxY - b.bounds.MinY

	// Windows on horizontal walls
	if height >= 4 {
		for x := b.bounds.MinX + 2; x <= b.bounds.MaxX-2; x += 2 {
			if b.entranceDir != North {
				g.Set(Point{x, b.bounds.MinY}, p.Window, false)
			}
			if b.entranceDir != South {
				g.Set(Point{x, b.bounds.MaxY}, p.Window, false)
			}
		}
	}

	// Windows on vertical walls
	if width >= 4 {
		for y := b.bounds.MinY + 2; y <= b.bounds.MaxY-2; y += 2 {
			if b.entranceDir != West {
				g.Set(Point{b.bounds.MinX, y}, p.Window, false)
			}
			if b.entranceDir != East {
				g.Set(Point{b.bounds.MaxX, y}, p.Window, false)
			}
		}
	}

	// Place door on entrance side
	g.Set(doorPos, p.Door, true)
}

func (b *Building) getDoorPosition() Point {
	center := b.bounds.Center()
	switch b.entranceDir {
	case North:
		return Point{center.X, b.bounds.MinY}
	case South:
		return Point{center.X, b.bounds.MaxY}
	case East:
		return Point{b.bounds.MaxX, center.Y}
	case West:
		return Point{b.bounds.MinX, center.Y}
	}
	return Point{center.X, b.bounds.MaxY}
}

func (b *Building) GetBounds() Bounds { return b.bounds }
func (b *Building) GetAnchors() []Anchor {
	door := b.getDoorPosition()
	dx, dy := b.entranceDir.Delta()
	// Anchor is one tile outside the door
	return []Anchor{{Position: door.Add(dx, dy), Direction: b.entranceDir.Opposite()}}
}
func (b *Building) GetZone() *Zone { return b.zone }

// Cabin creates a small rustic structure with chimney
type Cabin struct {
	bounds      Bounds
	entranceDir Direction
	zone        *Zone
}

func NewCabin(bounds Bounds, entranceDir Direction, zone *Zone) *Cabin {
	return &Cabin{bounds: bounds, entranceDir: entranceDir, zone: zone}
}

func (c *Cabin) Render(g *Grid, p *Palette) {
	// Fill interior with wood floor
	interior := Bounds{c.bounds.MinX + 1, c.bounds.MinY + 1, c.bounds.MaxX - 1, c.bounds.MaxY - 1}
	g.Rect(interior, p.WoodFloor, true)

	// Draw wood walls
	g.RectOutline(c.bounds, p.WoodWall, false)

	// Add chimney on top-right corner (outside the building)
	chimneyPos := Point{c.bounds.MaxX, c.bounds.MinY - 1}
	if chimneyPos.Y >= 0 {
		g.Set(chimneyPos, p.Chimney, false)
	}

	// Place door
	doorPos := c.getDoorPosition()
	g.Set(doorPos, p.Door, true)
}

func (c *Cabin) getDoorPosition() Point {
	center := c.bounds.Center()
	switch c.entranceDir {
	case North:
		return Point{center.X, c.bounds.MinY}
	case South:
		return Point{center.X, c.bounds.MaxY}
	case East:
		return Point{c.bounds.MaxX, center.Y}
	case West:
		return Point{c.bounds.MinX, center.Y}
	}
	return Point{center.X, c.bounds.MaxY}
}

func (c *Cabin) GetBounds() Bounds { return c.bounds }
func (c *Cabin) GetAnchors() []Anchor {
	door := c.getDoorPosition()
	dx, dy := c.entranceDir.Delta()
	return []Anchor{{Position: door.Add(dx, dy), Direction: c.entranceDir.Opposite()}}
}
func (c *Cabin) GetZone() *Zone { return c.zone }

// Tower creates a larger central structure
type Tower struct {
	center      Point
	radius      int
	entranceDir Direction
	zone        *Zone
}

func NewTower(center Point, radius int, entranceDir Direction, zone *Zone) *Tower {
	return &Tower{center: center, radius: radius, entranceDir: entranceDir, zone: zone}
}

func (t *Tower) Render(g *Grid, p *Palette) {
	bounds := t.GetBounds()

	// Fill with cobblestone
	g.Rect(bounds, p.Cobblestone, true)

	// Draw walls
	g.RectOutline(bounds, p.Building, false)

	// Corner pillars for visual distinction
	g.Set(Point{bounds.MinX, bounds.MinY}, p.Pillar, false)
	g.Set(Point{bounds.MaxX, bounds.MinY}, p.Pillar, false)
	g.Set(Point{bounds.MinX, bounds.MaxY}, p.Pillar, false)
	g.Set(Point{bounds.MaxX, bounds.MaxY}, p.Pillar, false)

	// Inner chamber with star
	if t.radius >= 3 {
		inner := Bounds{t.center.X - 1, t.center.Y - 1, t.center.X + 1, t.center.Y + 1}
		g.RectOutline(inner, p.WhiteBuilding, false)
		g.Set(t.center, p.Star, true)
	}

	// Place door
	doorPos := t.getDoorPosition()
	g.Set(doorPos, p.Door, true)
}

func (t *Tower) getDoorPosition() Point {
	switch t.entranceDir {
	case North:
		return Point{t.center.X, t.center.Y - t.radius}
	case South:
		return Point{t.center.X, t.center.Y + t.radius}
	case East:
		return Point{t.center.X + t.radius, t.center.Y}
	case West:
		return Point{t.center.X - t.radius, t.center.Y}
	}
	return Point{t.center.X, t.center.Y + t.radius}
}

func (t *Tower) GetBounds() Bounds {
	return Bounds{t.center.X - t.radius, t.center.Y - t.radius,
		t.center.X + t.radius, t.center.Y + t.radius}
}

func (t *Tower) GetAnchors() []Anchor {
	door := t.getDoorPosition()
	dx, dy := t.entranceDir.Delta()
	return []Anchor{{Position: door.Add(dx, dy), Direction: t.entranceDir.Opposite()}}
}

func (t *Tower) GetZone() *Zone { return t.zone }

// Courtyard creates an enclosed open space with entrances
type Courtyard struct {
	bounds    Bounds
	wallStyle string
	entrances []Direction
	zone      *Zone
}

func NewCourtyard(bounds Bounds, wallStyle string, entrances []Direction, zone *Zone) *Courtyard {
	return &Courtyard{bounds: bounds, wallStyle: wallStyle, entrances: entrances, zone: zone}
}

func (c *Courtyard) Render(g *Grid, p *Palette) {
	wallTile := p.Building
	switch c.wallStyle {
	case "white":
		wallTile = p.WhiteBuilding
	case "wood":
		wallTile = p.WoodWall
	}

	// Fill interior
	interior := Bounds{c.bounds.MinX + 1, c.bounds.MinY + 1, c.bounds.MaxX - 1, c.bounds.MaxY - 1}
	g.Rect(interior, p.Cobblestone, true)

	// Draw walls
	g.RectOutline(c.bounds, wallTile, false)

	// Add corner pillars
	g.Set(Point{c.bounds.MinX, c.bounds.MinY}, p.Pillar, false)
	g.Set(Point{c.bounds.MaxX, c.bounds.MinY}, p.Pillar, false)
	g.Set(Point{c.bounds.MinX, c.bounds.MaxY}, p.Pillar, false)
	g.Set(Point{c.bounds.MaxX, c.bounds.MaxY}, p.Pillar, false)

	// Add fountain in center
	center := c.bounds.Center()
	g.Set(center, p.Water, false)
	g.Set(Point{center.X - 1, center.Y}, p.Sand, true)
	g.Set(Point{center.X + 1, center.Y}, p.Sand, true)
	g.Set(Point{center.X, center.Y - 1}, p.Sand, true)
	g.Set(Point{center.X, center.Y + 1}, p.Sand, true)

	// Place entrances (gates)
	for _, dir := range c.entrances {
		var gatePos Point
		switch dir {
		case North:
			gatePos = Point{center.X, c.bounds.MinY}
		case South:
			gatePos = Point{center.X, c.bounds.MaxY}
		case East:
			gatePos = Point{c.bounds.MaxX, center.Y}
		case West:
			gatePos = Point{c.bounds.MinX, center.Y}
		}
		g.Set(gatePos, p.Door, true)
	}
}

func (c *Courtyard) GetBounds() Bounds { return c.bounds }

func (c *Courtyard) GetAnchors() []Anchor {
	anchors := make([]Anchor, len(c.entrances))
	center := c.bounds.Center()
	for i, dir := range c.entrances {
		var pos Point
		switch dir {
		case North:
			pos = Point{center.X, c.bounds.MinY - 1}
		case South:
			pos = Point{center.X, c.bounds.MaxY + 1}
		case East:
			pos = Point{c.bounds.MaxX + 1, center.Y}
		case West:
			pos = Point{c.bounds.MinX - 1, center.Y}
		}
		anchors[i] = Anchor{Position: pos, Direction: dir.Opposite()}
	}
	return anchors
}

func (c *Courtyard) GetZone() *Zone { return c.zone }

// Shrine creates a small sacred/special space
type Shrine struct {
	center Point
	size   int // 1 = 3x3, 2 = 5x5, etc.
	zone   *Zone
}

func NewShrine(center Point, size int, zone *Zone) *Shrine {
	return &Shrine{center: center, size: size, zone: zone}
}

func (s *Shrine) Render(g *Grid, p *Palette) {
	bounds := s.GetBounds()

	// Cobblestone base
	g.Rect(bounds, p.Cobblestone, true)

	// Marker border
	g.RectOutline(bounds, p.Marker, true)

	// Star center
	g.Set(s.center, p.Star, true)
}

func (s *Shrine) GetBounds() Bounds {
	return Bounds{s.center.X - s.size, s.center.Y - s.size,
		s.center.X + s.size, s.center.Y + s.size}
}

func (s *Shrine) GetAnchors() []Anchor {
	return []Anchor{
		{Position: Point{s.center.X, s.center.Y + s.size + 1}, Direction: North},
	}
}

func (s *Shrine) GetZone() *Zone { return s.zone }

// ---- Infrastructure Components ----

// Plaza creates a cobblestone gathering area
type Plaza struct {
	center Point
	radius int
	shape  string // "square" or "circle"
}

func NewPlaza(center Point, radius int, shape string) *Plaza {
	return &Plaza{center: center, radius: radius, shape: shape}
}

func (pl *Plaza) Render(g *Grid, p *Palette) {
	if pl.shape == "circle" {
		for dy := -pl.radius; dy <= pl.radius; dy++ {
			for dx := -pl.radius; dx <= pl.radius; dx++ {
				if dx*dx+dy*dy <= pl.radius*pl.radius {
					g.Set(Point{pl.center.X + dx, pl.center.Y + dy}, p.Cobblestone, true)
				}
			}
		}
	} else {
		bounds := pl.GetBounds()
		g.Rect(bounds, p.Cobblestone, true)
	}
}

func (pl *Plaza) GetBounds() Bounds {
	return Bounds{pl.center.X - pl.radius, pl.center.Y - pl.radius,
		pl.center.X + pl.radius, pl.center.Y + pl.radius}
}

func (pl *Plaza) GetAnchors() []Anchor {
	return []Anchor{
		{Position: Point{pl.center.X, pl.center.Y - pl.radius}, Direction: South},
		{Position: Point{pl.center.X, pl.center.Y + pl.radius}, Direction: North},
		{Position: Point{pl.center.X - pl.radius, pl.center.Y}, Direction: East},
		{Position: Point{pl.center.X + pl.radius, pl.center.Y}, Direction: West},
	}
}

func (pl *Plaza) GetZone() *Zone { return nil }

// Dock extends into water from shore
type Dock struct {
	origin    Point
	direction Direction
	length    int
	width     int
	zone      *Zone
}

func NewDock(origin Point, direction Direction, length, width int, zone *Zone) *Dock {
	return &Dock{origin: origin, direction: direction, length: length, width: width, zone: zone}
}

func (d *Dock) Render(g *Grid, p *Palette) {
	dx, dy := d.direction.Delta()
	halfWidth := d.width / 2

	for i := 0; i < d.length; i++ {
		baseX := d.origin.X + dx*i
		baseY := d.origin.Y + dy*i

		// Draw width perpendicular to direction
		for w := -halfWidth; w <= halfWidth; w++ {
			var px, py int
			if dx != 0 { // Moving horizontally, width is vertical
				px, py = baseX, baseY+w
			} else { // Moving vertically, width is horizontal
				px, py = baseX+w, baseY
			}
			g.Set(Point{px, py}, p.Dock, true)
		}
	}
}

func (d *Dock) GetBounds() Bounds {
	dx, dy := d.direction.Delta()
	endX := d.origin.X + dx*(d.length-1)
	endY := d.origin.Y + dy*(d.length-1)
	halfWidth := d.width / 2

	minX, maxX := min(d.origin.X, endX), max(d.origin.X, endX)
	minY, maxY := min(d.origin.Y, endY), max(d.origin.Y, endY)

	if dx != 0 {
		minY -= halfWidth
		maxY += halfWidth
	} else {
		minX -= halfWidth
		maxX += halfWidth
	}

	return Bounds{minX, minY, maxX, maxY}
}

func (d *Dock) GetAnchors() []Anchor {
	return []Anchor{{Position: d.origin, Direction: d.direction.Opposite()}}
}

func (d *Dock) GetZone() *Zone { return d.zone }

// Bridge spans non-walkable terrain
type Bridge struct {
	start, end Point
}

func NewBridge(start, end Point) *Bridge {
	return &Bridge{start: start, end: end}
}

func (br *Bridge) Render(g *Grid, p *Palette) {
	g.Line(br.start, br.end, p.Bridge, true)
}

func (br *Bridge) GetBounds() Bounds {
	return Bounds{
		min(br.start.X, br.end.X), min(br.start.Y, br.end.Y),
		max(br.start.X, br.end.X), max(br.start.Y, br.end.Y),
	}
}

func (br *Bridge) GetAnchors() []Anchor {
	return []Anchor{
		{Position: br.start, Direction: South},
		{Position: br.end, Direction: North},
	}
}

func (br *Bridge) GetZone() *Zone { return nil }

// ---- Decoration Components ----

// Scatter is a generic decoration placer
type ScatterDecor struct {
	bounds  Bounds
	tile    string
	density float64
	rng     *RNG
}

func NewScatterDecor(bounds Bounds, tile string, density float64, rng *RNG) *ScatterDecor {
	return &ScatterDecor{bounds: bounds, tile: tile, density: density, rng: rng}
}

func (s *ScatterDecor) Render(g *Grid, p *Palette) {
	g.Scatter(s.bounds, s.tile, false, s.density, s.rng, nil)
}

func (s *ScatterDecor) GetBounds() Bounds  { return s.bounds }
func (s *ScatterDecor) GetAnchors() []Anchor { return nil }
func (s *ScatterDecor) GetZone() *Zone       { return nil }

// Border draws a decorative ring around a bounds
type Border struct {
	inner    Bounds
	tile     string
	walkable bool
}

func NewBorder(inner Bounds, tile string, walkable bool) *Border {
	return &Border{inner: inner, tile: tile, walkable: walkable}
}

func (b *Border) Render(g *Grid, p *Palette) {
	outer := b.inner.Expand(1)
	g.RectOutline(outer, b.tile, b.walkable)
}

func (b *Border) GetBounds() Bounds    { return b.inner.Expand(1) }
func (b *Border) GetAnchors() []Anchor { return nil }
func (b *Border) GetZone() *Zone       { return nil }

// Signpost marks a path exit with destination info
type Signpost struct {
	position  Point
	direction Direction
	zone      *Zone
}

func NewSignpost(position Point, direction Direction, destName, destHint string) *Signpost {
	return &Signpost{
		position:  position,
		direction: direction,
		zone: &Zone{
			Name:        "Signpost",
			Description: destHint,
			Bounds:      Bounds{position.X - 1, position.Y - 1, position.X + 1, position.Y + 1},
		},
	}
}

func (s *Signpost) Render(g *Grid, p *Palette) {
	// Place marker at signpost position
	g.Set(s.position, p.Marker, true)
}

func (s *Signpost) GetBounds() Bounds {
	return Bounds{s.position.X, s.position.Y, s.position.X, s.position.Y}
}
func (s *Signpost) GetAnchors() []Anchor { return nil }
func (s *Signpost) GetZone() *Zone       { return s.zone }

// Pond creates a small water feature
type Pond struct {
	center Point
	radius int
}

func NewPond(center Point, radius int) *Pond {
	return &Pond{center: center, radius: radius}
}

func (p *Pond) Render(g *Grid, pal *Palette) {
	// Water center
	for dy := -p.radius + 1; dy < p.radius; dy++ {
		for dx := -p.radius + 1; dx < p.radius; dx++ {
			if dx*dx+dy*dy < (p.radius-1)*(p.radius-1) {
				g.Set(Point{p.center.X + dx, p.center.Y + dy}, pal.Water, false)
			}
		}
	}
	// Sand border
	for dy := -p.radius; dy <= p.radius; dy++ {
		for dx := -p.radius; dx <= p.radius; dx++ {
			dist := dx*dx + dy*dy
			if dist >= (p.radius-1)*(p.radius-1) && dist <= p.radius*p.radius {
				pt := Point{p.center.X + dx, p.center.Y + dy}
				if g.Get(pt) != pal.Water {
					g.Set(pt, pal.Sand, true)
				}
			}
		}
	}
}

func (p *Pond) GetBounds() Bounds {
	return Bounds{p.center.X - p.radius, p.center.Y - p.radius,
		p.center.X + p.radius, p.center.Y + p.radius}
}
func (p *Pond) GetAnchors() []Anchor { return nil }
func (p *Pond) GetZone() *Zone       { return nil }

// Garden creates an organized vegetation area
type Garden struct {
	bounds Bounds
	rng    *RNG
}

func NewGarden(bounds Bounds, rng *RNG) *Garden {
	return &Garden{bounds: bounds, rng: rng}
}

func (g *Garden) Render(grid *Grid, p *Palette) {
	// Cobblestone border
	grid.RectOutline(g.bounds, p.Cobblestone, true)

	// Fill with grass
	interior := Bounds{g.bounds.MinX + 1, g.bounds.MinY + 1, g.bounds.MaxX - 1, g.bounds.MaxY - 1}
	grid.Rect(interior, p.Grass, true)

	// Scatter bushes in a pattern
	for y := interior.MinY; y <= interior.MaxY; y++ {
		for x := interior.MinX; x <= interior.MaxX; x++ {
			if (x+y)%3 == 0 && g.rng.Float64() < 0.5 {
				grid.Set(Point{x, y}, p.Bush, false)
			}
		}
	}
}

func (g *Garden) GetBounds() Bounds    { return g.bounds }
func (g *Garden) GetAnchors() []Anchor { return nil }
func (g *Garden) GetZone() *Zone       { return nil }

// Ruins creates a partially destroyed structure
type Ruins struct {
	bounds Bounds
	decay  float64 // 0.0-1.0, how much is destroyed
	rng    *RNG
}

func NewRuins(bounds Bounds, decay float64, rng *RNG) *Ruins {
	return &Ruins{bounds: bounds, decay: decay, rng: rng}
}

func (r *Ruins) Render(g *Grid, p *Palette) {
	// Place walls with gaps based on decay
	for x := r.bounds.MinX; x <= r.bounds.MaxX; x++ {
		for y := r.bounds.MinY; y <= r.bounds.MaxY; y++ {
			isEdge := x == r.bounds.MinX || x == r.bounds.MaxX ||
			          y == r.bounds.MinY || y == r.bounds.MaxY
			if isEdge {
				if r.rng.Float64() > r.decay {
					g.Set(Point{x, y}, p.Building, false)
				} else {
					g.Set(Point{x, y}, p.Cobblestone, true)
				}
			} else {
				g.Set(Point{x, y}, p.Cobblestone, true)
			}
		}
	}
}

func (r *Ruins) GetBounds() Bounds    { return r.bounds }
func (r *Ruins) GetAnchors() []Anchor {
	center := r.bounds.Center()
	return []Anchor{{Position: Point{center.X, r.bounds.MaxY + 1}, Direction: North}}
}
func (r *Ruins) GetZone() *Zone { return nil }

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
