package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"dconn.dev/internal/services"
)

// ProjectHandler handles project-related endpoints
type ProjectHandler struct {
	projectService *services.ProjectService
}

// NewProjectHandler creates a new ProjectHandler
func NewProjectHandler(ps *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{projectService: ps}
}

// ListProjects handles GET /api/projects
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects := h.projectService.GetAll()
	respondJSON(w, http.StatusOK, projects)
}

// GetProject handles GET /api/projects/{id}
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, err := h.projectService.GetByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	respondJSON(w, http.StatusOK, project)
}
