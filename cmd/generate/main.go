package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"

	"dconn.dev/internal/generation"
)

var minimum, maximum int = 10000, 99999

// WorldConfig defines the entire world layout
var worldConfig = []generation.ChunkConfig{
	{
		ChunkX:      0,
		ChunkY:      0,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeGrassland,
		Shorelines:  []generation.Direction{},
		Connections: []generation.Direction{generation.North, generation.South, generation.East, generation.West},
		SignpostHints: map[generation.Direction]string{
			generation.North: "Clockwork echoes drift from the north. The machines await.",
			generation.South: "Castle spires glimmer in the distance.",
			generation.East:  "The smell of salt and sea beckons.",
			generation.West:  "Shadows dance between ancient trees, and mountains loom beyond.",
		},
	},
	{
		ChunkX:      0,
		ChunkY:      -1,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeUrban,
		Shorelines:  []generation.Direction{generation.North},
		Connections: []generation.Direction{generation.South},
		SignpostHints: map[generation.Direction]string{
			generation.South: "The peaceful starting meadows lie beyond.",
		},
	},
	{
		ChunkX:      -1,
		ChunkY:      -1,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeMountain,
		Shorelines:  []generation.Direction{generation.West, generation.North, generation.East},
		Connections: []generation.Direction{generation.South},
		SignpostHints: map[generation.Direction]string{
			generation.South: "The forest whispers of tools and crafts below.",
		},
	},
	{
		ChunkX:      -1,
		ChunkY:      0,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeForest,
		Shorelines:  []generation.Direction{generation.West},
		Connections: []generation.Direction{generation.North, generation.South, generation.East},
		SignpostHints: map[generation.Direction]string{
			generation.North: "The mountains hold secrets of transformation.",
			generation.South: "Scholars gather where knowledge flows freely.",
			generation.East:  "The central isle lies just beyond.",
		},
	},
	{
		ChunkX:      1,
		ChunkY:      0,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeCoastal,
		Shorelines:  []generation.Direction{generation.East},
		Connections: []generation.Direction{generation.West, generation.South},
		SignpostHints: map[generation.Direction]string{
			generation.West:  "Return to the peaceful starting meadows.",
			generation.South: "Towers of healing rise to the south.",
		},
	},
	{
		ChunkX:      -1,
		ChunkY:      1,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeUrban,
		Shorelines:  []generation.Direction{generation.West, generation.South},
		Connections: []generation.Direction{generation.North, generation.East},
		SignpostHints: map[generation.Direction]string{
			generation.North: "Deep woods hide workshops of craft.",
			generation.East:  "Games and glory await at the castle!",
		},
	},
	{
		ChunkX:      0,
		ChunkY:      1,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeCastle,
		Shorelines:  []generation.Direction{generation.South},
		Connections: []generation.Direction{generation.North, generation.West, generation.East},
		SignpostHints: map[generation.Direction]string{
			generation.North: "The peaceful starting isle awaits.",
			generation.West:  "Seekers of knowledge head this way.",
			generation.East:  "Healers tend to the tower beyond.",
		},
	},
	{
		ChunkX:      1,
		ChunkY:      1,
		Seed:        uint64(rand.IntN(maximum-minimum+1) + minimum),
		Biome:       generation.BiomeUrban,
		Shorelines:  []generation.Direction{generation.East, generation.South},
		Connections: []generation.Direction{generation.North, generation.West},
		SignpostHints: map[generation.Direction]string{
			generation.North: "Salty breezes drift from the harbor.",
			generation.West:  "The castle's games echo across the land.",
		},
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: generate <output-dir>")
		fmt.Println("       generate <output-dir> <chunk-x> <chunk-y>  (generate single chunk)")
		os.Exit(1)
	}

	outputDir := os.Args[1]

	// Populate each chunk's project placements from data/placements.json,
	// validating against data/projects.json so the map can't drift from the
	// project list.
	if err := applyPlacements(worldConfig, outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Ensure output directory exists
	chunksDir := filepath.Join(outputDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate chunks. Track failures so a partial regeneration exits non-zero
	// rather than silently leaving stale chunk files on disk (and, via the
	// Makefile, restarting the service with mismatched data).
	var failures []string
	for _, config := range worldConfig {
		fmt.Printf("Generating chunk (%d, %d) - %s biome...\n", config.ChunkX, config.ChunkY, config.Biome)

		gen := generation.NewChunkGenerator(&config)
		chunk, err := gen.Generate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %v\n", err)
			failures = append(failures, fmt.Sprintf("(%d, %d): %v", config.ChunkX, config.ChunkY, err))
			continue
		}

		// Write to file
		filename := fmt.Sprintf("%d_%d.json", config.ChunkX, config.ChunkY)
		path := filepath.Join(chunksDir, filename)

		data, err := json.MarshalIndent(chunk, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR marshaling JSON: %v\n", err)
			failures = append(failures, fmt.Sprintf("(%d, %d): %v", config.ChunkX, config.ChunkY, err))
			continue
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR writing file: %v\n", err)
			failures = append(failures, fmt.Sprintf("(%d, %d): %v", config.ChunkX, config.ChunkY, err))
			continue
		}

		fmt.Printf("  Created %s (%d zones)\n", filename, len(chunk.Zones))
	}

	// Build the signpost registry: one seam marker per shared border, owned by
	// the South/East chunk, carrying the hint for both directions of travel.
	if err := writeSignposts(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR writing signposts: %v\n", err)
		failures = append(failures, fmt.Sprintf("signposts: %v", err))
	}

	if len(failures) > 0 {
		fmt.Fprintf(os.Stderr, "\nFAILED: %d chunk(s) not regenerated (existing files left unchanged):\n  - %s\n",
			len(failures), strings.Join(failures, "\n  - "))
		os.Exit(1)
	}

	fmt.Println("Done!")
}

// dirName maps a Direction to the travel-direction key used in the registry.
var dirName = map[generation.Direction]string{
	generation.North: "north",
	generation.South: "south",
	generation.East:  "east",
	generation.West:  "west",
}

// writeSignposts derives the global signpost registry from worldConfig and
// writes it to <outputDir>/signposts.json. Each shared border ("seam") yields
// exactly one entry, owned by the chunk that holds the South/East connection,
// pairing that chunk's outbound hint with the neighbor's reciprocal hint.
func writeSignposts(outputDir string) error {
	const size = generation.ChunkSize

	configByCoord := make(map[[2]int]*generation.ChunkConfig, len(worldConfig))
	for i := range worldConfig {
		c := &worldConfig[i]
		configByCoord[[2]int{c.ChunkX, c.ChunkY}] = c
	}

	hintOr := func(s string) string {
		if s == "" {
			return "A path leads onward..."
		}
		return s
	}

	registry := generation.SignpostRegistry{}
	for i := range worldConfig {
		owner := &worldConfig[i]
		for _, dir := range owner.Connections {
			var local [2]int
			var axis string
			var partition int
			switch dir {
			case generation.South:
				local = [2]int{size / 2, size - 1} // (25, 49)
				axis = "horizontal"
				partition = (owner.ChunkY + 1) * size
			case generation.East:
				local = [2]int{size - 1, size / 2} // (49, 25)
				axis = "vertical"
				partition = (owner.ChunkX + 1) * size
			default:
				continue // North/West seams are owned by the neighbor
			}

			ndx, ndy := dir.Delta()
			neighborCoord := [2]int{owner.ChunkX + ndx, owner.ChunkY + ndy}
			neighbor, ok := configByCoord[neighborCoord]
			if !ok {
				fmt.Fprintf(os.Stderr, "  WARNING: chunk (%d,%d) connects %s but no neighbor at (%d,%d); skipping seam marker\n",
					owner.ChunkX, owner.ChunkY, dirName[dir], neighborCoord[0], neighborCoord[1])
				continue
			}

			rev := dir.Opposite()
			registry.Signposts = append(registry.Signposts, generation.SignpostEntry{
				Chunk:     [2]int{owner.ChunkX, owner.ChunkY},
				Local:     local,
				World:     [2]int{owner.ChunkX*size + local[0], owner.ChunkY*size + local[1]},
				Axis:      axis,
				Partition: partition,
				Hints: map[string]string{
					dirName[dir]: hintOr(owner.SignpostHints[dir]),
					dirName[rev]: hintOr(neighbor.SignpostHints[rev]),
				},
			})
		}
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}
	path := filepath.Join(outputDir, "signposts.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	fmt.Printf("Wrote signposts.json (%d seam markers)\n", len(registry.Signposts))
	return nil
}
