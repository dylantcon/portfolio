package generation

import "fmt"

// NodeType identifies what kind of node this is in the graph
type NodeType int

const (
	NodeEdgePort   NodeType = iota // Entry/exit point at chunk border
	NodeComponent                  // A placed component (building, shrine, etc.)
	NodeHub                        // Central connection point
)

// Node represents a connectable element in the chunk graph
type Node struct {
	ID       string
	Type     NodeType
	Position Point    // Primary position (center or anchor point)
	Anchors  []Anchor // Connection points for paths
	Bounds   Bounds   // Space occupied by this node
	Zone     *Zone    // Optional zone data if this node has a project
}

// Edge represents a connection between two nodes
type Edge struct {
	From, To string  // Node IDs
	Weight   float64 // Cost/distance (for MST calculation)
	Required bool    // Must this edge exist for validity?
	Path     []Point // Realized path on the grid (filled during rendering)
}

// Graph manages nodes and edges for chunk generation
type Graph struct {
	Nodes map[string]*Node
	Edges []*Edge

	// Adjacency list for quick lookups
	Adjacent map[string][]string
}

// NewGraph creates an empty graph
func NewGraph() *Graph {
	return &Graph{
		Nodes:    make(map[string]*Node),
		Edges:    make([]*Edge, 0),
		Adjacent: make(map[string][]string),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(n *Node) {
	g.Nodes[n.ID] = n
	if g.Adjacent[n.ID] == nil {
		g.Adjacent[n.ID] = make([]string, 0)
	}
}

// AddEdge adds an edge between two nodes
func (g *Graph) AddEdge(fromID, toID string, required bool) error {
	from, ok := g.Nodes[fromID]
	if !ok {
		return fmt.Errorf("node %s not found", fromID)
	}
	to, ok := g.Nodes[toID]
	if !ok {
		return fmt.Errorf("node %s not found", toID)
	}

	// Calculate weight as Manhattan distance between positions
	weight := manhattanDist(from.Position, to.Position)

	edge := &Edge{
		From:     fromID,
		To:       toID,
		Weight:   float64(weight),
		Required: required,
	}

	g.Edges = append(g.Edges, edge)
	g.Adjacent[fromID] = append(g.Adjacent[fromID], toID)
	g.Adjacent[toID] = append(g.Adjacent[toID], fromID)

	return nil
}

// GetEdge returns the edge between two nodes if it exists
func (g *Graph) GetEdge(fromID, toID string) *Edge {
	for _, e := range g.Edges {
		if (e.From == fromID && e.To == toID) || (e.From == toID && e.To == fromID) {
			return e
		}
	}
	return nil
}

// IsConnected checks if all nodes are reachable from a starting node using BFS
func (g *Graph) IsConnected(startID string) bool {
	if len(g.Nodes) == 0 {
		return true
	}

	visited := make(map[string]bool)
	queue := []string{startID}
	visited[startID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighborID := range g.Adjacent[current] {
			if !visited[neighborID] {
				visited[neighborID] = true
				queue = append(queue, neighborID)
			}
		}
	}

	return len(visited) == len(g.Nodes)
}

// FindUnreachable returns nodes not reachable from the start node
func (g *Graph) FindUnreachable(startID string) []string {
	visited := make(map[string]bool)
	queue := []string{startID}
	visited[startID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighborID := range g.Adjacent[current] {
			if !visited[neighborID] {
				visited[neighborID] = true
				queue = append(queue, neighborID)
			}
		}
	}

	unreachable := make([]string, 0)
	for id := range g.Nodes {
		if !visited[id] {
			unreachable = append(unreachable, id)
		}
	}
	return unreachable
}

// GetEdgePorts returns all nodes that are edge ports
func (g *Graph) GetEdgePorts() []*Node {
	ports := make([]*Node, 0)
	for _, n := range g.Nodes {
		if n.Type == NodeEdgePort {
			ports = append(ports, n)
		}
	}
	return ports
}

// GetProjectNodes returns all nodes that have an associated project
func (g *Graph) GetProjectNodes() []*Node {
	projects := make([]*Node, 0)
	for _, n := range g.Nodes {
		if n.Zone != nil && n.Zone.ProjectID != "" {
			projects = append(projects, n)
		}
	}
	return projects
}

// MST computes a minimum spanning tree using Kruskal's algorithm
// Returns the edges that form the MST
func (g *Graph) MST() []*Edge {
	// Union-Find data structure
	parent := make(map[string]string)
	rank := make(map[string]int)

	var find func(x string) string
	find = func(x string) string {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(x, y string) bool {
		rootX, rootY := find(x), find(y)
		if rootX == rootY {
			return false
		}
		if rank[rootX] < rank[rootY] {
			rootX, rootY = rootY, rootX
		}
		parent[rootY] = rootX
		if rank[rootX] == rank[rootY] {
			rank[rootX]++
		}
		return true
	}

	// Initialize union-find
	for id := range g.Nodes {
		parent[id] = id
		rank[id] = 0
	}

	// Sort edges by weight (simple insertion sort for small graphs)
	sortedEdges := make([]*Edge, len(g.Edges))
	copy(sortedEdges, g.Edges)
	for i := 1; i < len(sortedEdges); i++ {
		for j := i; j > 0 && sortedEdges[j].Weight < sortedEdges[j-1].Weight; j-- {
			sortedEdges[j], sortedEdges[j-1] = sortedEdges[j-1], sortedEdges[j]
		}
	}

	mst := make([]*Edge, 0)
	for _, edge := range sortedEdges {
		if union(edge.From, edge.To) {
			mst = append(mst, edge)
		}
	}

	return mst
}

func manhattanDist(a, b Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}
