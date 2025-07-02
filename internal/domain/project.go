package domain

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Goal        string    `json:"goal"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func NewProject(name, description, goal string) *Project {
	now := time.Now()
	return &Project{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Goal:        goal,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

type ProjectRepository interface {
	Create(project *Project) error
	Get(id string) (*Project, error)
	List() ([]*Project, error)
	SetCurrent(id string) error
	GetCurrent() (*Project, error)
}