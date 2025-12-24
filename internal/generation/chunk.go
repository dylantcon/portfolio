package generation

import (
	"fmt"
)

const ChunkSize = 50

// ChunkGenerator generates chunk data from configuration
type ChunkGenerator struct {
	config  *ChunkConfig
	grid    *Grid
	graph   *Graph
	palette *Palette
	biome   *Biome
	rng     *RNG

	components      []Component // Structural components (rendered before paths)
	terrainFeatures []Component // Terrain features (rendered after paths)
	zones           []*Zone
}

// NewChunkGenerator creates a generator for the given config
func NewChunkGenerator(config *ChunkConfig) *ChunkGenerator {
	return &ChunkGenerator{
		config:          config,
		palette:         DefaultPalette(),
		biome:           GetBiome(config.Biome),
		rng:             NewRNG(config.Seed),
		components:      make([]Component, 0),
		terrainFeatures: make([]Component, 0),
		zones:           make([]*Zone, 0),
	}
}

// Generate produces the chunk definition
func (cg *ChunkGenerator) Generate() (*ChunkDefinition, error) {
	// 1. Initialize grid with base terrain
	cg.initGrid()

	// 2. Build the connectivity graph
	cg.buildGraph()

	// 3. Place edge terrain (shorelines, mountains)
	cg.placeTerrain()

	// 4. Place project structures
	if err := cg.placeProjects(); err != nil {
		return nil, fmt.Errorf("placing projects: %w", err)
	}

	// 5. Create central hub if we have multiple connections
	cg.placeHub()

	// 6. Place signposts at exits
	cg.placeSignposts()

	// 7. Render structural components (buildings, terrain edges)
	cg.renderComponents()

	// 8. Route paths between graph nodes
	if err := cg.routePaths(); err != nil {
		return nil, fmt.Errorf("routing paths: %w", err)
	}

	// 9. Add terrain features AFTER paths (so they don't block routes)
	cg.placeTerrainFeatures()
	cg.renderTerrainFeatures()

	// 10. Add decoration (trees, bushes)
	cg.addDecoration()

	// 11. Validate accessibility
	if err := cg.validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 12. Build output
	return cg.buildOutput(), nil
}

func (cg *ChunkGenerator) initGrid() {
	cg.grid = NewGrid(ChunkSize, ChunkSize, cg.biome.BaseTile, cg.biome.BaseWalkable)
}

func (cg *ChunkGenerator) buildGraph() {
	cg.graph = NewGraph()

	// Add edge port nodes for each connection
	for _, dir := range cg.config.Connections {
		port := cg.createEdgePort(dir)
		cg.graph.AddNode(port)
	}
}

func (cg *ChunkGenerator) createEdgePort(dir Direction) *Node {
	var pos Point
	mid := ChunkSize / 2

	switch dir {
	case North:
		pos = Point{mid, 0}
	case South:
		pos = Point{mid, ChunkSize - 1}
	case East:
		pos = Point{ChunkSize - 1, mid}
	case West:
		pos = Point{0, mid}
	}

	return &Node{
		ID:       fmt.Sprintf("port_%d", dir),
		Type:     NodeEdgePort,
		Position: pos,
		Anchors:  []Anchor{{Position: pos, Direction: dir.Opposite()}},
		Bounds:   Bounds{pos.X, pos.Y, pos.X, pos.Y},
	}
}

func (cg *ChunkGenerator) placeTerrain() {
	// Place shorelines
	for _, dir := range cg.config.Shorelines {
		shore := NewShoreline(dir, 3, 2, ChunkSize)
		cg.components = append(cg.components, shore)
	}

	// Mountain biome gets mountains along the top/northwest
	if cg.config.Biome == BiomeMountain {
		// Place mountains in upper-left, leaving passes for connections
		passes := make([]Point, 0)
		// Add a pass in the middle for traversal
		passes = append(passes, Point{15, 10})

		mtns := NewMountainRange(
			Bounds{3, 3, 25, 12},
			passes,
			2,
		)
		cg.components = append(cg.components, mtns)
	}
}

func (cg *ChunkGenerator) placeProjects() error {
	if len(cg.config.Projects) == 0 {
		return nil
	}

	// Calculate placement positions based on number of projects
	positions := cg.calculateProjectPositions(len(cg.config.Projects))

	for i, proj := range cg.config.Projects {
		pos := positions[i]

		zone := &Zone{
			Name:        proj.Name,
			Description: proj.Description,
			ProjectID:   proj.ProjectID,
		}

		var comp Component

		// Create structure based on type
		switch proj.Structure {
		case "tower":
			radius := 3 + proj.Size
			entranceDir := cg.findBestEntrance(pos, radius)
			comp = NewTower(pos, radius, entranceDir, zone)

		case "shrine":
			size := proj.Size
			comp = NewShrine(pos, size, zone)

		case "courtyard":
			size := 4 + proj.Size*2
			bounds := Bounds{pos.X - size, pos.Y - size, pos.X + size, pos.Y + size}
			entrances := []Direction{South} // Default entrance
			comp = NewCourtyard(bounds, "stone", entrances, zone)

		case "cabin":
			size := 3 + proj.Size
			bounds := Bounds{pos.X - size, pos.Y - size/2, pos.X + size, pos.Y + size/2}
			entranceDir := cg.findBestEntrance(pos, size)
			comp = NewCabin(bounds, entranceDir, zone)

		default: // "building"
			size := 3 + proj.Size
			bounds := Bounds{pos.X - size, pos.Y - size/2, pos.X + size, pos.Y + size/2}
			entranceDir := cg.findBestEntrance(pos, size)
			comp = NewBuilding(bounds, "stone", entranceDir, zone)
		}

		// Update zone bounds from component
		zone.Bounds = comp.GetBounds()

		cg.components = append(cg.components, comp)
		cg.zones = append(cg.zones, zone)

		// Add to graph
		node := &Node{
			ID:       fmt.Sprintf("project_%s", proj.ProjectID),
			Type:     NodeComponent,
			Position: pos,
			Anchors:  comp.GetAnchors(),
			Bounds:   comp.GetBounds(),
			Zone:     zone,
		}
		cg.graph.AddNode(node)
	}

	return nil
}

func (cg *ChunkGenerator) calculateProjectPositions(count int) []Point {
	positions := make([]Point, count)

	// Calculate safe bounds (avoid shorelines)
	minX, minY := 10, 10
	maxX, maxY := ChunkSize-10, ChunkSize-10

	for _, dir := range cg.config.Shorelines {
		switch dir {
		case North:
			minY = 15
		case South:
			maxY = ChunkSize - 15
		case East:
			maxX = ChunkSize - 15
		case West:
			minX = 15
		}
	}

	// Also avoid mountain areas in mountain biome
	if cg.config.Biome == BiomeMountain {
		minY = max(minY, 20) // Mountains take up top portion
	}

	// Calculate center of safe area
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2
	safeWidth := maxX - minX
	safeHeight := maxY - minY

	if count == 1 {
		positions[0] = Point{centerX, centerY}
		return positions
	}

	if count == 2 {
		offsetX := safeWidth / 4
		offsetY := safeHeight / 4
		positions[0] = Point{centerX - offsetX, centerY - offsetY}
		positions[1] = Point{centerX + offsetX, centerY + offsetY}
		return positions
	}

	if count == 3 {
		offsetX := safeWidth / 3
		offsetY := safeHeight / 3
		positions[0] = Point{centerX, centerY - offsetY}
		positions[1] = Point{centerX - offsetX, centerY + offsetY/2}
		positions[2] = Point{centerX + offsetX, centerY + offsetY/2}
		return positions
	}

	if count == 4 {
		offsetX := safeWidth / 3
		offsetY := safeHeight / 3
		positions[0] = Point{centerX - offsetX, centerY - offsetY}
		positions[1] = Point{centerX + offsetX, centerY - offsetY}
		positions[2] = Point{centerX - offsetX, centerY + offsetY}
		positions[3] = Point{centerX + offsetX, centerY + offsetY}
		return positions
	}

	// For 5+ projects, distribute in safe area
	radius := min(safeWidth, safeHeight) / 3
	for i := 0; i < count; i++ {
		angle := float64(i) * (6.28318 / float64(count))
		dx := int(float64(radius) * cos(angle))
		dy := int(float64(radius) * sin(angle))
		positions[i] = Point{centerX + dx, centerY + dy}
	}

	return positions
}

func (cg *ChunkGenerator) findBestEntrance(pos Point, size int) Direction {
	center := Point{ChunkSize / 2, ChunkSize / 2}

	// Entrance should face toward center of chunk
	dx := center.X - pos.X
	dy := center.Y - pos.Y

	if abs(dx) > abs(dy) {
		if dx > 0 {
			return East
		}
		return West
	}
	if dy > 0 {
		return South
	}
	return North
}

func (cg *ChunkGenerator) placeHub() {
	// If we have multiple connections or projects, add a central hub
	if len(cg.config.Connections) > 1 || len(cg.config.Projects) > 0 {
		center := Point{ChunkSize / 2, ChunkSize / 2}
		plaza := NewPlaza(center, 3, "square")
		cg.components = append(cg.components, plaza)

		hubNode := &Node{
			ID:       "hub",
			Type:     NodeHub,
			Position: center,
			Anchors:  plaza.GetAnchors(),
			Bounds:   plaza.GetBounds(),
		}
		cg.graph.AddNode(hubNode)

		// Connect hub to all edge ports
		for _, dir := range cg.config.Connections {
			portID := fmt.Sprintf("port_%d", dir)
			cg.graph.AddEdge("hub", portID, true)
		}

		// Connect hub to all project nodes
		for _, proj := range cg.config.Projects {
			projID := fmt.Sprintf("project_%s", proj.ProjectID)
			cg.graph.AddEdge("hub", projID, true)
		}
	} else if len(cg.config.Connections) == 1 {
		// Single connection - just mark path from edge to interior
		// Paths will be routed later
	}
}

func (cg *ChunkGenerator) placeSignposts() {
	// Add signposts ON the path near edge connections
	for _, dir := range cg.config.Connections {
		hint := cg.config.SignpostHints[dir]
		if hint == "" {
			hint = "A path leads onward..."
		}

		// Position signpost ON the path, a few tiles in from edge
		var pos Point
		offset := 4
		mid := ChunkSize / 2

		switch dir {
		case North:
			pos = Point{mid, offset} // On the north path
		case South:
			pos = Point{mid, ChunkSize - 1 - offset} // On the south path
		case East:
			pos = Point{ChunkSize - 1 - offset, mid} // On the east path
		case West:
			pos = Point{offset, mid} // On the west path
		}

		signpost := NewSignpost(pos, dir, "", hint)
		cg.components = append(cg.components, signpost)
		cg.zones = append(cg.zones, signpost.GetZone())
	}
}

func (cg *ChunkGenerator) placeTerrainFeatures() {
	// Add biome-specific terrain features in chunk interior
	// These are placed AFTER paths are routed so they don't block connectivity

	switch cg.config.Biome {
	case BiomeGrassland:
		// Add a pond in a corner
		if cg.rng.Float64() < 0.5 {
			pos := Point{10 + cg.rng.Intn(8), 38 + cg.rng.Intn(5)}
			pond := NewPond(pos, 3)
			cg.terrainFeatures = append(cg.terrainFeatures, pond)
		}

	case BiomeForest:
		// Add dense grove areas in corners (away from paths)
		grove1 := NewGrove(Bounds{5, 5, 12, 12}, 0.35, cg.palette.Tree, cg.rng)
		grove2 := NewGrove(Bounds{38, 38, 45, 45}, 0.35, cg.palette.Tree, cg.rng)
		cg.terrainFeatures = append(cg.terrainFeatures, grove1, grove2)

	case BiomeCoastal:
		// Add dock extending into water if we have east shoreline
		for _, dir := range cg.config.Shorelines {
			if dir == East {
				dock := NewDock(Point{ChunkSize - 8, ChunkSize / 2}, East, 5, 3, nil)
				cg.terrainFeatures = append(cg.terrainFeatures, dock)
			}
		}

	case BiomeUrban:
		// Add a garden
		garden := NewGarden(Bounds{8, 38, 15, 45}, cg.rng)
		cg.terrainFeatures = append(cg.terrainFeatures, garden)

	case BiomeCastle:
		// Add ruins in a corner
		ruins := NewRuins(Bounds{38, 5, 44, 10}, 0.4, cg.rng)
		cg.terrainFeatures = append(cg.terrainFeatures, ruins)

	case BiomeMountain:
		// Add a small pine grove in the lower portion
		grove := NewGrove(Bounds{35, 38, 42, 45}, 0.2, cg.palette.PineTree, cg.rng)
		cg.terrainFeatures = append(cg.terrainFeatures, grove)
	}
}

func (cg *ChunkGenerator) renderTerrainFeatures() {
	for _, feat := range cg.terrainFeatures {
		feat.Render(cg.grid, cg.palette)
	}
}

func (cg *ChunkGenerator) renderComponents() {
	for _, comp := range cg.components {
		comp.Render(cg.grid, cg.palette)
	}
}

func (cg *ChunkGenerator) routePaths() error {
	// Build set of points to avoid (component interiors)
	avoid := make(map[Point]bool)
	for _, comp := range cg.components {
		bounds := comp.GetBounds()
		for y := bounds.MinY; y <= bounds.MaxY; y++ {
			for x := bounds.MinX; x <= bounds.MaxX; x++ {
				// Don't avoid anchors
				isAnchor := false
				for _, a := range comp.GetAnchors() {
					if a.Position.X == x && a.Position.Y == y {
						isAnchor = true
						break
					}
				}
				if !isAnchor && !cg.grid.IsWalkable(Point{x, y}) {
					avoid[Point{x, y}] = true
				}
			}
		}
	}

	// Route each edge
	for _, edge := range cg.graph.Edges {
		fromNode := cg.graph.Nodes[edge.From]
		toNode := cg.graph.Nodes[edge.To]

		// Find best anchors to connect
		fromAnchor := cg.findClosestAnchor(fromNode, toNode.Position)
		toAnchor := cg.findClosestAnchor(toNode, fromNode.Position)

		// Find path
		path := cg.grid.FindPathAvoid(fromAnchor, toAnchor, avoid)
		if path == nil {
			// Try without avoidance for required edges
			if edge.Required {
				path = cg.grid.FindPath(fromAnchor, toAnchor, nil)
			}
		}

		if path == nil && edge.Required {
			return fmt.Errorf("could not route required path from %s to %s", edge.From, edge.To)
		}

		if path != nil {
			edge.Path = path
			// Draw path on grid
			for _, p := range path {
				if cg.grid.Get(p) == cg.palette.Grass || cg.grid.Get(p) == cg.palette.Sand {
					cg.grid.Set(p, cg.palette.Path, true)
				}
			}
		}
	}

	return nil
}

func (cg *ChunkGenerator) findClosestAnchor(node *Node, target Point) Point {
	if len(node.Anchors) == 0 {
		return node.Position
	}

	closest := node.Anchors[0].Position
	minDist := manhattanDist(closest, target)

	for _, a := range node.Anchors[1:] {
		d := manhattanDist(a.Position, target)
		if d < minDist {
			minDist = d
			closest = a.Position
		}
	}

	return closest
}

func (cg *ChunkGenerator) addDecoration() {
	// Get bounds to avoid (paths, structures)
	avoid := make(map[Point]bool)
	for y := 0; y < ChunkSize; y++ {
		for x := 0; x < ChunkSize; x++ {
			tile := cg.grid.Get(Point{x, y})
			if tile != cg.palette.Grass {
				avoid[Point{x, y}] = true
			}
		}
	}

	fullBounds := Bounds{0, 0, ChunkSize - 1, ChunkSize - 1}

	// Add trees
	if cg.biome.TreeDensity > 0 {
		cg.grid.Scatter(fullBounds, cg.biome.TreeType, false, cg.biome.TreeDensity, cg.rng, avoid)
	}

	// Add bushes
	if cg.biome.BushDensity > 0 {
		cg.grid.Scatter(fullBounds, cg.palette.Bush, false, cg.biome.BushDensity, cg.rng, avoid)
	}
}

func (cg *ChunkGenerator) validate() error {
	// Check that all project zones are reachable from at least one edge port
	if len(cg.graph.GetEdgePorts()) == 0 {
		// No edge ports means this is an isolated chunk (not really valid)
		return fmt.Errorf("chunk has no edge connections")
	}

	// Use first edge port as starting point
	ports := cg.graph.GetEdgePorts()
	startID := ports[0].ID

	// Check graph connectivity
	if !cg.graph.IsConnected(startID) {
		unreachable := cg.graph.FindUnreachable(startID)
		return fmt.Errorf("unreachable nodes: %v", unreachable)
	}

	// Flood fill from each edge port to verify actual tile accessibility
	for _, port := range ports {
		reachable := cg.floodFillReachable(port.Position)

		// Check each project zone is reachable
		for _, zone := range cg.zones {
			zoneReachable := false
			// Check if any anchor point of the zone is reachable
			for _, comp := range cg.components {
				if comp.GetZone() == zone {
					for _, anchor := range comp.GetAnchors() {
						if reachable[anchor.Position] {
							zoneReachable = true
							break
						}
					}
				}
			}
			if zoneReachable {
				continue
			}

			// Check if zone center is reachable
			center := zone.Bounds.Center()
			if reachable[center] {
				continue
			}

			return fmt.Errorf("zone %q not reachable from port %s", zone.Name, port.ID)
		}
	}

	return nil
}

func (cg *ChunkGenerator) floodFillReachable(start Point) map[Point]bool {
	reachable := make(map[Point]bool)
	queue := []Point{start}

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		if reachable[p] {
			continue
		}
		if !cg.grid.InBounds(p) {
			continue
		}
		if !cg.grid.IsWalkable(p) {
			continue
		}

		reachable[p] = true

		for _, adj := range p.Adjacent() {
			if !reachable[adj] {
				queue = append(queue, adj)
			}
		}
	}

	return reachable
}

func (cg *ChunkGenerator) buildOutput() *ChunkDefinition {
	// Convert zones to output format
	zoneDefs := make([]ZoneDef, len(cg.zones))
	for i, z := range cg.zones {
		zoneDefs[i] = ZoneDef{
			Name:        z.Name,
			Description: z.Description,
			Bounds: BoundsDef{
				MinX: z.Bounds.MinX,
				MaxX: z.Bounds.MaxX,
				MinY: z.Bounds.MinY,
				MaxY: z.Bounds.MaxY,
			},
			ProjectID: z.ProjectID,
		}
	}

	return &ChunkDefinition{
		Tiles: cg.grid.Tiles,
		Zones: zoneDefs,
	}
}

// Simple trig for position calculation (avoiding math import for these)
func cos(x float64) float64 {
	// Taylor series approximation, good enough for our purposes
	x = mod2pi(x)
	return 1 - x*x/2 + x*x*x*x/24 - x*x*x*x*x*x/720
}

func sin(x float64) float64 {
	x = mod2pi(x)
	return x - x*x*x/6 + x*x*x*x*x/120 - x*x*x*x*x*x*x/5040
}

func mod2pi(x float64) float64 {
	const twoPi = 6.283185307179586
	for x < 0 {
		x += twoPi
	}
	for x >= twoPi {
		x -= twoPi
	}
	return x
}
