package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusPlanned    TaskStatus = "planned"
	StatusInProgress TaskStatus = "in-progress"
	StatusCompleted  TaskStatus = "completed"
	StatusBlocked    TaskStatus = "blocked"
	StatusOnHold     TaskStatus = "on-hold"
	StatusCanceled   TaskStatus = "canceled"
)

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
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
	Priority    Priority   `json:"priority"`
	Parent      *string    `json:"parent,omitempty"`
	Children    []string   `json:"children,omitempty"`
	Labels      []string   `json:"labels,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	EstimatedHours *float64 `json:"estimatedHours,omitempty"`
	ActualHours    *float64 `json:"actualHours,omitempty"`
	AssignedTo     *string  `json:"assignedTo,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Verification *CompletionVerification `json:"verification,omitempty"`
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

type VerificationEvidence struct {
	ID              string    `json:"id"`
	Evidence        string    `json:"evidence"`                    // Agent's memo notes on what was tested
	TestedAt        time.Time `json:"testedAt"`                   // When verification occurred
	CommitHash      string    `json:"commitHash,omitempty"`       // Git state during test
	TestType        string    `json:"testType,omitempty"`         // Type of test performed
	TestResults     string    `json:"testResults,omitempty"`      // Detailed results or output
	FilesAffected   []string  `json:"filesAffected,omitempty"`    // Files tested/modified
	RelatedCriteria []int     `json:"relatedCriteria,omitempty"`  // Loose mapping to acceptance criteria indices
}

type CompletionVerification struct {
	CompletedBy     string                 `json:"completedBy,omitempty"`
	CompletedAt     time.Time             `json:"completedAt"`
	Evidence        []VerificationEvidence `json:"evidence"`
	CompletionNotes string                `json:"completionNotes,omitempty"`
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
			Priority:    PriorityMedium,
			Children:    make([]string, 0),
			Labels:      make([]string, 0),
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
	ProjectID    *string
	Status       *TaskStatus
	Priority     *Priority
	Parent       *string
	Labels       []string
	AssignedTo   *string
	DueBefore    *time.Time
	DueAfter     *time.Time
	CreatedAfter *time.Time
	UpdatedAfter *time.Time
}

type TaskRepository interface {
	Create(task *Task) error
	Update(id string, updates map[string]interface{}) (*Task, error)
	Get(id string) (*Task, error)
	List(filter TaskFilter) ([]*Task, error)
	Delete(id string) error
}

// Helper methods for TODO functionality

// NewTODO creates a task optimized for TODO management
func NewTODO(projectID, title, description string, priority Priority) *Task {
	task := NewTask(projectID, title, description)
	task.Card.Priority = priority
	return task
}

// IsOverdue checks if the task is past its due date
func (t *Task) IsOverdue() bool {
	if t.Card.DueDate == nil {
		return false
	}
	return time.Now().After(*t.Card.DueDate) && t.Card.Status != StatusCompleted
}

// DaysUntilDue returns the number of days until the task is due (negative if overdue)
func (t *Task) DaysUntilDue() *int {
	if t.Card.DueDate == nil {
		return nil
	}
	days := int(t.Card.DueDate.Sub(time.Now()).Hours() / 24)
	return &days
}

// Complete marks the task as completed and sets completion time (legacy method)
func (t *Task) Complete() {
	t.Card.Status = StatusCompleted
	now := time.Now()
	t.Card.CompletedAt = &now
	t.Card.UpdatedAt = now
}

// CompleteWithVerification marks the task as completed with verification evidence
func (t *Task) CompleteWithVerification(evidence []VerificationEvidence, completedBy, completionNotes string) error {
	// Validate that we have at least one evidence per acceptance criteria
	if len(evidence) == 0 {
		return fmt.Errorf("verification evidence is required for task completion")
	}
	
	if len(t.Criteria.Acceptance) > 0 && len(evidence) < len(t.Criteria.Acceptance) {
		return fmt.Errorf("insufficient verification evidence: need at least %d evidence items for %d acceptance criteria", len(t.Criteria.Acceptance), len(t.Criteria.Acceptance))
	}
	
	// Assign UUIDs to evidence items if not provided
	for i := range evidence {
		if evidence[i].ID == "" {
			evidence[i].ID = uuid.New().String()
		}
		if evidence[i].TestedAt.IsZero() {
			evidence[i].TestedAt = time.Now()
		}
	}
	
	// Mark task as completed
	t.Card.Status = StatusCompleted
	now := time.Now()
	t.Card.CompletedAt = &now
	t.Card.UpdatedAt = now
	
	// Store verification data
	t.Card.Verification = &CompletionVerification{
		CompletedBy:     completedBy,
		CompletedAt:     now,
		Evidence:        evidence,
		CompletionNotes: completionNotes,
	}
	
	return nil
}

// Reopen reopens a completed task
func (t *Task) Reopen() {
	if t.Card.Status == StatusCompleted {
		t.Card.Status = StatusPlanned
		t.Card.CompletedAt = nil
		t.Card.UpdatedAt = time.Now()
	}
}

// AddLabel adds a label to the task if it doesn't already exist
func (t *Task) AddLabel(label string) {
	for _, l := range t.Card.Labels {
		if l == label {
			return
		}
	}
	t.Card.Labels = append(t.Card.Labels, label)
	t.Card.UpdatedAt = time.Now()
}

// RemoveLabel removes a label from the task
func (t *Task) RemoveLabel(label string) {
	for i, l := range t.Card.Labels {
		if l == label {
			t.Card.Labels = append(t.Card.Labels[:i], t.Card.Labels[i+1:]...)
			t.Card.UpdatedAt = time.Now()
			break
		}
	}
}

// HasLabel checks if the task has a specific label
func (t *Task) HasLabel(label string) bool {
	for _, l := range t.Card.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// SetDueDate sets the due date for the task
func (t *Task) SetDueDate(dueDate time.Time) {
	t.Card.DueDate = &dueDate
	t.Card.UpdatedAt = time.Now()
}

// ClearDueDate removes the due date from the task
func (t *Task) ClearDueDate() {
	t.Card.DueDate = nil
	t.Card.UpdatedAt = time.Now()
}

// UpdateProgress updates actual hours worked
func (t *Task) UpdateProgress(hours float64) {
	if t.Card.ActualHours == nil {
		t.Card.ActualHours = &hours
	} else {
		*t.Card.ActualHours += hours
	}
	t.Card.UpdatedAt = time.Now()
}

// GetProgressPercentage returns progress as a percentage (actual/estimated * 100)
func (t *Task) GetProgressPercentage() *float64 {
	if t.Card.EstimatedHours == nil || *t.Card.EstimatedHours == 0 {
		return nil
	}
	if t.Card.ActualHours == nil {
		zero := 0.0
		return &zero
	}
	percentage := (*t.Card.ActualHours / *t.Card.EstimatedHours) * 100
	return &percentage
}