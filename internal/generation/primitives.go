package generation

import (
	"container/heap"
	"math"
)

// Rect fills a rectangular area with a tile
func (g *Grid) Rect(b Bounds, tile string, walkable bool) {
	for y := b.MinY; y <= b.MaxY; y++ {
		for x := b.MinX; x <= b.MaxX; x++ {
			g.Set(Point{x, y}, tile, walkable)
		}
	}
}

// RectOutline draws the outline of a rectangle
func (g *Grid) RectOutline(b Bounds, tile string, walkable bool) {
	for x := b.MinX; x <= b.MaxX; x++ {
		g.Set(Point{x, b.MinY}, tile, walkable)
		g.Set(Point{x, b.MaxY}, tile, walkable)
	}
	for y := b.MinY; y <= b.MaxY; y++ {
		g.Set(Point{b.MinX, y}, tile, walkable)
		g.Set(Point{b.MaxX, y}, tile, walkable)
	}
}

// Line draws a line between two points using Bresenham's algorithm
func (g *Grid) Line(from, to Point, tile string, walkable bool) {
	dx := abs(to.X - from.X)
	dy := -abs(to.Y - from.Y)
	sx := 1
	if from.X > to.X {
		sx = -1
	}
	sy := 1
	if from.Y > to.Y {
		sy = -1
	}
	err := dx + dy

	x, y := from.X, from.Y
	for {
		g.Set(Point{x, y}, tile, walkable)
		if x == to.X && y == to.Y {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
}

// FloodFill fills an area starting from a point, replacing matching tiles
func (g *Grid) FloodFill(start Point, newTile string, walkable bool) {
	if !g.InBounds(start) {
		return
	}

	oldTile := g.Get(start)
	if oldTile == newTile {
		return
	}

	queue := []Point{start}
	visited := make(map[Point]bool)

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		if visited[p] || !g.InBounds(p) {
			continue
		}
		if g.Get(p) != oldTile {
			continue
		}

		visited[p] = true
		g.Set(p, newTile, walkable)

		for _, adj := range p.Adjacent() {
			if !visited[adj] {
				queue = append(queue, adj)
			}
		}
	}
}

// Scatter randomly places tiles within bounds at a given density
func (g *Grid) Scatter(b Bounds, tile string, walkable bool, density float64, rng *RNG, avoid map[Point]bool) {
	for y := b.MinY; y <= b.MaxY; y++ {
		for x := b.MinX; x <= b.MaxX; x++ {
			p := Point{x, y}
			if avoid != nil && avoid[p] {
				continue
			}
			if rng.Float64() < density {
				g.Set(p, tile, walkable)
			}
		}
	}
}

// ScatterOnTile randomly places tiles on top of a specific existing tile
func (g *Grid) ScatterOnTile(b Bounds, targetTile, newTile string, walkable bool, density float64, rng *RNG) {
	for y := b.MinY; y <= b.MaxY; y++ {
		for x := b.MinX; x <= b.MaxX; x++ {
			p := Point{x, y}
			if g.Get(p) == targetTile && rng.Float64() < density {
				g.Set(p, newTile, walkable)
			}
		}
	}
}

// ---- A* Pathfinding ----

// astarNode represents a node in the A* priority queue
type astarNode struct {
	point    Point
	gScore   float64 // Cost from start
	fScore   float64 // gScore + heuristic
	parent   *Point
	index    int
}

// priorityQueue implements heap.Interface for A*
type priorityQueue []*astarNode

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].fScore < pq[j].fScore }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x interface{}) {
	n := x.(*astarNode)
	n.index = len(*pq)
	*pq = append(*pq, n)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := old[len(old)-1]
	old[len(old)-1] = nil
	*pq = old[:len(old)-1]
	return n
}

// FindPath uses A* to find a path between two points
// walkableOverride allows treating certain non-walkable tiles as walkable (for path carving)
// Returns nil if no path found
func (g *Grid) FindPath(from, to Point, walkableOverride map[Point]bool) []Point {
	if !g.InBounds(from) || !g.InBounds(to) {
		return nil
	}

	isWalkable := func(p Point) bool {
		if walkableOverride != nil && walkableOverride[p] {
			return true
		}
		return g.IsWalkable(p)
	}

	// A* implementation
	openSet := &priorityQueue{}
	heap.Init(openSet)

	gScore := make(map[Point]float64)
	cameFrom := make(map[Point]Point)
	inOpen := make(map[Point]bool)

	gScore[from] = 0
	startNode := &astarNode{
		point:  from,
		gScore: 0,
		fScore: heuristic(from, to),
	}
	heap.Push(openSet, startNode)
	inOpen[from] = true

	for openSet.Len() > 0 {
		current := heap.Pop(openSet).(*astarNode)
		delete(inOpen, current.point)

		if current.point == to {
			// Reconstruct path
			path := []Point{to}
			curr := to
			for curr != from {
				curr = cameFrom[curr]
				path = append([]Point{curr}, path...)
			}
			return path
		}

		for _, neighbor := range current.point.Adjacent() {
			if !g.InBounds(neighbor) {
				continue
			}
			if !isWalkable(neighbor) && neighbor != to {
				continue
			}

			tentativeG := gScore[current.point] + 1

			if oldG, exists := gScore[neighbor]; !exists || tentativeG < oldG {
				cameFrom[neighbor] = current.point
				gScore[neighbor] = tentativeG
				fScore := tentativeG + heuristic(neighbor, to)

				if !inOpen[neighbor] {
					heap.Push(openSet, &astarNode{
						point:  neighbor,
						gScore: tentativeG,
						fScore: fScore,
					})
					inOpen[neighbor] = true
				}
			}
		}
	}

	return nil // No path found
}

// FindPathAvoid finds a path while avoiding certain points
func (g *Grid) FindPathAvoid(from, to Point, avoid map[Point]bool) []Point {
	if !g.InBounds(from) || !g.InBounds(to) {
		return nil
	}

	isWalkable := func(p Point) bool {
		if avoid != nil && avoid[p] {
			return false
		}
		return g.IsWalkable(p)
	}

	// Same A* but with avoid check
	openSet := &priorityQueue{}
	heap.Init(openSet)

	gScore := make(map[Point]float64)
	cameFrom := make(map[Point]Point)
	inOpen := make(map[Point]bool)

	gScore[from] = 0
	heap.Push(openSet, &astarNode{point: from, gScore: 0, fScore: heuristic(from, to)})
	inOpen[from] = true

	for openSet.Len() > 0 {
		current := heap.Pop(openSet).(*astarNode)
		delete(inOpen, current.point)

		if current.point == to {
			path := []Point{to}
			curr := to
			for curr != from {
				curr = cameFrom[curr]
				path = append([]Point{curr}, path...)
			}
			return path
		}

		for _, neighbor := range current.point.Adjacent() {
			if !g.InBounds(neighbor) || (!isWalkable(neighbor) && neighbor != to) {
				continue
			}

			tentativeG := gScore[current.point] + 1
			if oldG, exists := gScore[neighbor]; !exists || tentativeG < oldG {
				cameFrom[neighbor] = current.point
				gScore[neighbor] = tentativeG
				if !inOpen[neighbor] {
					heap.Push(openSet, &astarNode{
						point:  neighbor,
						gScore: tentativeG,
						fScore: tentativeG + heuristic(neighbor, to),
					})
					inOpen[neighbor] = true
				}
			}
		}
	}

	return nil
}

func heuristic(a, b Point) float64 {
	return math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---- Seeded RNG ----

// RNG is a simple seeded random number generator (LCG)
type RNG struct {
	state uint64
}

// NewRNG creates a new RNG with the given seed
func NewRNG(seed uint64) *RNG {
	return &RNG{state: seed}
}

// Uint64 returns a pseudo-random uint64
func (r *RNG) Uint64() uint64 {
	// LCG parameters from Numerical Recipes
	r.state = r.state*6364136223846793005 + 1442695040888963407
	return r.state
}

// Float64 returns a pseudo-random float64 in [0, 1)
func (r *RNG) Float64() float64 {
	return float64(r.Uint64()>>11) / (1 << 53)
}

// Intn returns a pseudo-random int in [0, n)
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.Uint64() % uint64(n))
}

// IntRange returns a pseudo-random int in [min, max]
func (r *RNG) IntRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + r.Intn(max-min+1)
}

// Choice returns a random element from a slice
func (r *RNG) Choice(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[r.Intn(len(items))]
}

// Shuffle randomly reorders a slice of points
func (r *RNG) Shuffle(points []Point) {
	for i := len(points) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		points[i], points[j] = points[j], points[i]
	}
}
