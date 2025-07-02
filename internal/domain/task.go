package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusPlanned    TaskStatus = "planned"
	StatusInProgress TaskStatus = "in-progress"
	StatusCompleted  TaskStatus = "completed"
	StatusBlocked    TaskStatus = "blocked"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

type Task struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Card      Card      `json:"card"`
	Context   Context   `json:"context"`
	Criteria  Criteria  `json:"criteria"`
}

type Card struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Parent      *string    `json:"parent,omitempty"`
	Children    []string   `json:"children,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Context struct {
	Files            []string   `json:"files"`
	Dependencies     []string   `json:"dependencies"`
	Assumptions      []string   `json:"assumptions"`
	Blockers         []string   `json:"blockers"`
	Decisions        []string   `json:"decisions"`
	ContextualHeader string     `json:"contextualHeader,omitempty"`
	LastVerified     time.Time  `json:"lastVerified"`
	Confidence       Confidence `json:"confidence"`
}

type Criteria struct {
	Acceptance    []string `json:"acceptance"`
	Verification  []string `json:"verification"`
	TestScenarios []string `json:"testScenarios,omitempty"`
}

func NewTask(projectID, title, description string) *Task {
	now := time.Now()
	return &Task{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Card: Card{
			Title:       title,
			Description: description,
			Status:      StatusPlanned,
			Children:    make([]string, 0),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Context: Context{
			Files:        make([]string, 0),
			Dependencies: make([]string, 0),
			Assumptions:  make([]string, 0),
			Blockers:     make([]string, 0),
			Decisions:    make([]string, 0),
			LastVerified: now,
			Confidence:   ConfidenceMedium,
		},
		Criteria: Criteria{
			Acceptance:    make([]string, 0),
			Verification:  make([]string, 0),
			TestScenarios: make([]string, 0),
		},
	}
}

type TaskFilter struct {
	ProjectID *string
	Status    *TaskStatus
	Parent    *string
}

type TaskRepository interface {
	Create(task *Task) error
	Update(id string, updates map[string]interface{}) (*Task, error)
	Get(id string) (*Task, error)
	List(filter TaskFilter) ([]*Task, error)
	Delete(id string) error
}