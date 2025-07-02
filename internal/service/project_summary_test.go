package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rcliao/compass/internal/domain"
	"github.com/rcliao/compass/internal/storage"
)

func TestProjectSummaryService_GenerateProjectSummary(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := NewTaskService(memStorage)
	projectService := NewProjectService(memStorage)
	planningService := NewPlanningService(memStorage, taskService, projectService)
	summaryService := NewProjectSummaryService(taskService, projectService, planningService)
	
	// Create a project
	project := domain.NewProject("Test Project", "A test project", "Build awesome software")
	err := projectService.Create(project)
	require.NoError(t, err)
	
	// Create diverse tasks
	task1 := domain.NewTask(project.ID, "Completed Task", "A completed task")
	task1.Card.Status = domain.StatusCompleted
	task1.Context.Confidence = domain.ConfidenceHigh
	
	task2 := domain.NewTask(project.ID, "In Progress Task", "A task in progress")
	task2.Card.Status = domain.StatusInProgress
	task2.Context.Confidence = domain.ConfidenceMedium
	
	task3 := domain.NewTask(project.ID, "Blocked Task", "A blocked task")
	task3.Card.Status = domain.StatusBlocked
	task3.Context.Confidence = domain.ConfidenceLow
	task3.Context.Blockers = []string{"Waiting for API design"}
	
	task4 := domain.NewTask(project.ID, "Planned Task", "A planned task")
	task4.Card.Status = domain.StatusPlanned
	task4.Context.Confidence = domain.ConfidenceHigh
	task4.Criteria.Acceptance = []string{"Should work correctly", "Should be tested"}
	
	// Save tasks
	err = taskService.Create(task1)
	require.NoError(t, err)
	err = taskService.Create(task2)
	require.NoError(t, err)
	err = taskService.Create(task3)
	require.NoError(t, err)
	err = taskService.Create(task4)
	require.NoError(t, err)
	
	// Record some discoveries and decisions
	_, err = planningService.RecordDiscovery(project.ID, "Users prefer dark mode", domain.ImpactHigh, domain.SourceResearch, []string{task1.ID})
	require.NoError(t, err)
	_, err = planningService.RecordDecision(project.ID, "Framework choice", "React", "Better ecosystem", []string{"Vue", "Angular"}, true, []string{task2.ID})
	require.NoError(t, err)
	
	// Generate project summary
	summary, err := summaryService.GenerateProjectSummary(project.ID)
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	
	// Verify project info
	assert.Equal(t, project.ID, summary.Project.ID)
	assert.Equal(t, "Test Project", summary.Project.Name)
	
	// Verify task summary
	assert.Equal(t, 4, summary.TaskSummary.Total)
	assert.Equal(t, 1, summary.TaskSummary.ByStatus[domain.StatusCompleted])
	assert.Equal(t, 1, summary.TaskSummary.ByStatus[domain.StatusInProgress])
	assert.Equal(t, 1, summary.TaskSummary.ByStatus[domain.StatusBlocked])
	assert.Equal(t, 1, summary.TaskSummary.ByStatus[domain.StatusPlanned])
	
	assert.Equal(t, 2, summary.TaskSummary.ByConfidence[domain.ConfidenceHigh])
	assert.Equal(t, 1, summary.TaskSummary.ByConfidence[domain.ConfidenceMedium])
	assert.Equal(t, 1, summary.TaskSummary.ByConfidence[domain.ConfidenceLow])
	
	assert.Len(t, summary.TaskSummary.Blocked, 1)
	assert.Equal(t, task3.ID, summary.TaskSummary.Blocked[0].ID)
	
	assert.Len(t, summary.TaskSummary.Completed, 1)
	assert.Equal(t, task1.ID, summary.TaskSummary.Completed[0].ID)
	
	assert.Len(t, summary.TaskSummary.Recent, 4) // All tasks are recent
	
	// Verify discoveries and decisions
	assert.Len(t, summary.Discoveries, 1)
	assert.Equal(t, "Users prefer dark mode", summary.Discoveries[0].Insight)
	
	assert.Len(t, summary.Decisions, 1)
	assert.Equal(t, "Framework choice", summary.Decisions[0].Question)
	
	// Verify insights
	assert.NotNil(t, summary.Insights)
	assert.Equal(t, 1, summary.Insights.BlockerCount)
	assert.Equal(t, 1, summary.Insights.HighImpactDiscoveries)
	assert.NotEmpty(t, summary.Insights.VelocityTrend)
	assert.NotEmpty(t, summary.Insights.ContextHealth)
	assert.NotEmpty(t, summary.Insights.Recommendations)
	
	// Verify timestamp
	assert.False(t, summary.GeneratedAt.IsZero())
}

func TestProjectSummaryService_AnalyzeVelocityTrend(t *testing.T) {
	summaryService := &ProjectSummaryService{}
	
	// Test with no tasks
	trend := summaryService.analyzeVelocityTrend([]*domain.Task{})
	assert.Equal(t, "no_data", trend)
	
	// Test with tasks but no completed ones
	task1 := domain.NewTask("project-id", "Task 1", "A task")
	task1.Card.Status = domain.StatusPlanned
	
	trend = summaryService.analyzeVelocityTrend([]*domain.Task{task1})
	assert.Equal(t, "stable", trend)
}

func TestProjectSummaryService_AnalyzeContextHealth(t *testing.T) {
	summaryService := &ProjectSummaryService{}
	
	// Test with no tasks
	health := summaryService.analyzeContextHealth([]*domain.Task{})
	assert.Equal(t, "good", health)
	
	// Test with healthy tasks
	task1 := domain.NewTask("project-id", "Task 1", "A task")
	task1.Context.Confidence = domain.ConfidenceHigh
	task1.Criteria.Acceptance = []string{"Should work"}
	
	health = summaryService.analyzeContextHealth([]*domain.Task{task1})
	assert.Equal(t, "excellent", health)
	
	// Test with unhealthy tasks
	task2 := domain.NewTask("project-id", "Task 2", "Another task")
	task2.Context.Confidence = domain.ConfidenceLow
	task2.Criteria.Acceptance = []string{} // No acceptance criteria
	
	health = summaryService.analyzeContextHealth([]*domain.Task{task1, task2})
	assert.Contains(t, []string{"good", "fair", "poor"}, health) // Should be lower than excellent
}