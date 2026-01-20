package services

import (
	"fmt"

	"dconn.dev/internal/models"
)

// ProjectService handles project-related operations
type ProjectService struct {
	projects *models.ProjectList
}

// NewProjectService creates a new ProjectService
func NewProjectService(projects *models.ProjectList) *ProjectService {
	return &ProjectService{projects: projects}
}

// GetAll returns all projects
func (s *ProjectService) GetAll() []models.Project {
	return s.projects.Projects
}

// GetByID returns a specific project by ID
func (s *ProjectService) GetByID(id string) (*models.Project, error) {
	for i := range s.projects.Projects {
		if s.projects.Projects[i].ID == id {
			return &s.projects.Projects[i], nil
		}
	}
	return nil, fmt.Errorf("project not found: %s", id)
}
