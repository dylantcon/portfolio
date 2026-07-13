package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dconn.dev/internal/generation"
	"dconn.dev/internal/models"
)

// placementFile is the on-disk schema of data/placements.json.
type placementFile struct {
	Placements []placementEntry `json:"placements"`
}

// placementEntry maps a single project to its location and appearance on the map.
type placementEntry struct {
	ProjectID   string `json:"project_id"`
	Chunk       [2]int `json:"chunk"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Structure   string `json:"structure"`
	Size        int    `json:"size"`
}

// applyPlacements loads placements.json, validates it against projects.json and
// the world layout, then injects each placement into its chunk's config. It fails
// loudly if any project is unplaced, placed twice, or points at an unknown
// project/chunk, so the map can never silently drift from the project list.
func applyPlacements(configs []generation.ChunkConfig, dataDir string) error {
	placements, err := loadPlacements(dataDir)
	if err != nil {
		return err
	}
	projectIDs, err := loadProjectIDs(dataDir)
	if err != nil {
		return err
	}

	configByCoord := make(map[[2]int]*generation.ChunkConfig, len(configs))
	for i := range configs {
		c := &configs[i]
		configByCoord[[2]int{c.ChunkX, c.ChunkY}] = c
	}

	var problems []string
	placed := make(map[string]bool, len(placements))

	for _, p := range placements {
		switch {
		case p.ProjectID == "":
			problems = append(problems, "placement with empty project_id")
			continue
		case placed[p.ProjectID]:
			problems = append(problems, fmt.Sprintf("project %q has more than one placement", p.ProjectID))
			continue
		case !projectIDs[p.ProjectID]:
			problems = append(problems, fmt.Sprintf("placement references unknown project %q", p.ProjectID))
			continue
		}

		cfg, ok := configByCoord[p.Chunk]
		if !ok {
			problems = append(problems, fmt.Sprintf("project %q placed in nonexistent chunk (%d, %d)", p.ProjectID, p.Chunk[0], p.Chunk[1]))
			continue
		}

		placed[p.ProjectID] = true
		cfg.Projects = append(cfg.Projects, generation.ProjectPlacement{
			ProjectID:   p.ProjectID,
			Name:        p.Name,
			Description: p.Description,
			Structure:   p.Structure,
			Size:        p.Size,
		})
	}

	// Every project must be placed somewhere on the map.
	for id := range projectIDs {
		if !placed[id] {
			problems = append(problems, fmt.Sprintf("project %q has no map placement", id))
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("placement validation failed:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

func loadPlacements(dataDir string) ([]placementEntry, error) {
	path := filepath.Join(dataDir, "placements.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var pf placementFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return pf.Placements, nil
}

func loadProjectIDs(dataDir string) (map[string]bool, error) {
	path := filepath.Join(dataDir, "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var pl models.ProjectList
	if err := json.Unmarshal(data, &pl); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	ids := make(map[string]bool, len(pl.Projects))
	for _, p := range pl.Projects {
		ids[p.ID] = true
	}
	return ids, nil
}
