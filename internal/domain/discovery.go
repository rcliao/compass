package domain

import (
	"time"

	"github.com/google/uuid"
)

type Impact string

const (
	ImpactHigh   Impact = "high"
	ImpactMedium Impact = "medium"
	ImpactLow    Impact = "low"
)

type DiscoverySource string

const (
	SourceImplementation DiscoverySource = "implementation"
	SourceTesting        DiscoverySource = "testing"
	SourceResearch       DiscoverySource = "research"
	SourcePlanning       DiscoverySource = "planning"
)

type PlanningSessionStatus string

const (
	PlanningStatusActive    PlanningSessionStatus = "active"
	PlanningStatusCompleted PlanningSessionStatus = "completed"
	PlanningStatusAborted   PlanningSessionStatus = "aborted"
)

type PlanningSession struct {
	ID        string                `json:"id"`
	ProjectID string                `json:"projectId"`
	Name      string                `json:"name"`
	Status    PlanningSessionStatus `json:"status"`
	CreatedAt time.Time             `json:"createdAt"`
	Tasks     []string              `json:"tasks"`
}

type Discovery struct {
	ID            string          `json:"id"`
	ProjectID     string          `json:"projectId"`
	Timestamp     time.Time       `json:"timestamp"`
	Insight       string          `json:"insight"`
	Impact        Impact          `json:"impact"`
	AffectedTasks []string        `json:"affectedTasks"`
	Source        DiscoverySource `json:"source"`
}

type Decision struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"projectId"`
	Timestamp     time.Time `json:"timestamp"`
	Question      string    `json:"question"`
	Choice        string    `json:"choice"`
	Alternatives  []string  `json:"alternatives"`
	Rationale     string    `json:"rationale"`
	Reversible    bool      `json:"reversible"`
	AffectedTasks []string  `json:"affectedTasks"`
}

func NewPlanningSession(projectID, name string) *PlanningSession {
	return &PlanningSession{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Name:      name,
		Status:    PlanningStatusActive,
		CreatedAt: time.Now(),
		Tasks:     make([]string, 0),
	}
}

func NewDiscovery(projectID, insight string, impact Impact, source DiscoverySource) *Discovery {
	return &Discovery{
		ID:            uuid.New().String(),
		ProjectID:     projectID,
		Timestamp:     time.Now(),
		Insight:       insight,
		Impact:        impact,
		AffectedTasks: make([]string, 0),
		Source:        source,
	}
}

func NewDecision(projectID, question, choice, rationale string, alternatives []string, reversible bool) *Decision {
	return &Decision{
		ID:            uuid.New().String(),
		ProjectID:     projectID,
		Timestamp:     time.Now(),
		Question:      question,
		Choice:        choice,
		Alternatives:  alternatives,
		Rationale:     rationale,
		Reversible:    reversible,
		AffectedTasks: make([]string, 0),
	}
}