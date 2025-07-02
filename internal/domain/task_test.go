package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTask(t *testing.T) {
	projectID := "test-project-id"
	title := "Test Task"
	description := "This is a test task"

	task := NewTask(projectID, title, description)

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, projectID, task.ProjectID)
	assert.Equal(t, title, task.Card.Title)
	assert.Equal(t, description, task.Card.Description)
	assert.Equal(t, StatusPlanned, task.Card.Status)
	assert.NotZero(t, task.Card.CreatedAt)
	assert.NotZero(t, task.Card.UpdatedAt)
	assert.Equal(t, ConfidenceMedium, task.Context.Confidence)
	assert.NotNil(t, task.Context.Files)
	assert.NotNil(t, task.Context.Dependencies)
	assert.NotNil(t, task.Context.Assumptions)
	assert.NotNil(t, task.Context.Blockers)
	assert.NotNil(t, task.Context.Decisions)
	assert.NotNil(t, task.Criteria.Acceptance)
	assert.NotNil(t, task.Criteria.Verification)
	assert.NotNil(t, task.Criteria.TestScenarios)
}

func TestTaskStatus(t *testing.T) {
	statuses := []TaskStatus{StatusPlanned, StatusInProgress, StatusCompleted, StatusBlocked}
	
	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}
}

func TestConfidence(t *testing.T) {
	confidences := []Confidence{ConfidenceHigh, ConfidenceMedium, ConfidenceLow}
	
	for _, confidence := range confidences {
		assert.NotEmpty(t, string(confidence))
	}
}