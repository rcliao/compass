package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/rcliao/compass/internal/domain"
)

func TestHeaderGenerator_Generate(t *testing.T) {
	generator := NewHeaderGenerator(500) // Increase limit to see full header
	
	project := domain.NewProject("Test Project", "A test project", "Build awesome software")
	task := domain.NewTask(project.ID, "Implement authentication", "Add JWT-based authentication to the API")
	
	// Add some context
	task.Context.Files = []string{"auth.go", "middleware.go", "handlers.go"}
	task.Context.Dependencies = []string{"setup database", "create user model"}
	task.Context.Confidence = domain.ConfidenceHigh
	task.Criteria.Acceptance = []string{"Users can login", "JWT tokens are validated", "Protected routes work"}
	
	header := generator.Generate(task, project)
	
	assert.NotEmpty(t, header)
	assert.Contains(t, header, "Build awesome software")
	assert.Contains(t, header, "JWT-based authentication")
	assert.Contains(t, header, "Depends on:")
	assert.Contains(t, header, "Affects")
	assert.Contains(t, header, "Confidence: high")
	assert.Contains(t, header, "Has 3 acceptance criteria")
}

func TestHeaderGenerator_GenerateWithBlockers(t *testing.T) {
	generator := NewHeaderGenerator(200)
	
	project := domain.NewProject("Test Project", "A test project", "Build software")
	task := domain.NewTask(project.ID, "Deploy to production", "Deploy the application")
	
	task.Context.Blockers = []string{"Need SSL certificate", "Database migration not ready"}
	task.Card.Status = domain.StatusBlocked
	
	header := generator.Generate(task, project)
	
	assert.Contains(t, header, "Status: blocked")
	assert.Contains(t, header, "Blocked by:")
	assert.Contains(t, header, "SSL certificate")
}

func TestHeaderGenerator_UpdateTaskHeader(t *testing.T) {
	generator := NewHeaderGenerator(200)
	
	project := domain.NewProject("Test Project", "A test project", "Build software")
	task := domain.NewTask(project.ID, "Test task", "A test task")
	
	// Initially no header
	assert.Empty(t, task.Context.ContextualHeader)
	
	generator.UpdateTaskHeader(task, project)
	
	assert.NotEmpty(t, task.Context.ContextualHeader)
	assert.False(t, task.Context.LastVerified.IsZero())
}

func TestHeaderGenerator_IsStale(t *testing.T) {
	generator := NewHeaderGenerator(200)
	
	task := domain.NewTask("project-id", "Test task", "A test task")
	
	// Clear the LastVerified time (NewTask sets it to Now())
	task.Context.LastVerified = time.Time{}
	
	// New task should be stale (never verified)
	assert.True(t, generator.IsStale(task, time.Hour))
	
	// Update verification time
	task.Context.LastVerified = time.Now()
	assert.False(t, generator.IsStale(task, time.Hour))
	
	// Make it old
	task.Context.LastVerified = time.Now().Add(-2 * time.Hour)
	assert.True(t, generator.IsStale(task, time.Hour))
}

func TestHeaderGenerator_Truncate(t *testing.T) {
	generator := NewHeaderGenerator(200)
	
	// Short text should not be truncated
	short := "This is short"
	assert.Equal(t, short, generator.truncate(short, 50))
	
	// Long text should be truncated  
	long := "abcdefghijklmnopqrstuvwxyz1234567890"
	truncated := generator.truncate(long, 20)
	t.Logf("Original: %q (len=%d)", long, len(long))
	t.Logf("Truncated: %q (len=%d)", truncated, len(truncated))
	assert.True(t, len(truncated) <= 20, "Truncated length should not exceed limit")
	assert.True(t, len(truncated) < len(long), "Should be shorter than original")
	assert.Contains(t, truncated, "...", "Should contain ellipsis")
}

func TestHeaderGenerator_GenerateMinimal(t *testing.T) {
	generator := NewHeaderGenerator(200)
	
	// Minimal task with just title/description
	project := domain.NewProject("Test Project", "A test project", "")
	task := domain.NewTask(project.ID, "Simple task", "")
	
	header := generator.Generate(task, project)
	
	// Should still generate something meaningful
	assert.NotEmpty(t, header)
}