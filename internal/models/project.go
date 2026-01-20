package models

// Project represents a portfolio project
type Project struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	TechStack   []string `json:"tech_stack"`
	GitHubURL   string   `json:"github_url,omitempty"`
	LiveURL     string   `json:"live_url,omitempty"`
	Year        int      `json:"year"`
	Featured    bool     `json:"featured"`
}

// ProjectList wraps the array of projects
type ProjectList struct {
	Projects []Project `json:"projects"`
}
