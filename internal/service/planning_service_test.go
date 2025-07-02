package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rcliao/compass/internal/domain"
	"github.com/rcliao/compass/internal/storage"
)

func TestPlanningService_StartPlanningSession(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := NewTaskService(memStorage)
	projectService := NewProjectService(memStorage)
	planningService := NewPlanningService(memStorage, taskService, projectService)
	
	// Create a project first
	project := domain.NewProject("Test Project", "A test project", "Test planning")
	err := projectService.Create(project)
	require.NoError(t, err)
	
	// Start a planning session
	session, err := planningService.StartPlanningSession(project.ID, "Sprint Planning")
	assert.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "Sprint Planning", session.Name)
	assert.Equal(t, project.ID, session.ProjectID)
	assert.Equal(t, domain.PlanningStatusActive, session.Status)
	assert.NotEmpty(t, session.ID)
}

func TestPlanningService_RecordDiscovery(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := NewTaskService(memStorage)
	projectService := NewProjectService(memStorage)
	planningService := NewPlanningService(memStorage, taskService, projectService)
	
	// Create a project and task
	project := domain.NewProject("Test Project", "A test project", "Test planning")
	err := projectService.Create(project)
	require.NoError(t, err)
	
	task := domain.NewTask(project.ID, "Test Task", "A test task")
	err = taskService.Create(task)
	require.NoError(t, err)
	
	// Record a discovery
	discovery, err := planningService.RecordDiscovery(
		project.ID,
		"Users prefer OAuth over custom authentication",
		domain.ImpactHigh,
		domain.SourceResearch,
		[]string{task.ID},
	)
	
	assert.NoError(t, err)
	assert.NotNil(t, discovery)
	assert.Equal(t, "Users prefer OAuth over custom authentication", discovery.Insight)
	assert.Equal(t, domain.ImpactHigh, discovery.Impact)
	assert.Equal(t, domain.SourceResearch, discovery.Source)
	assert.Contains(t, discovery.AffectedTasks, task.ID)
	
	// Verify discovery is in the list
	discoveries, err := planningService.ListDiscoveries(project.ID)
	assert.NoError(t, err)
	assert.Len(t, discoveries, 1)
	assert.Equal(t, discovery.ID, discoveries[0].ID)
}

func TestPlanningService_RecordDecision(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := NewTaskService(memStorage)
	projectService := NewProjectService(memStorage)
	planningService := NewPlanningService(memStorage, taskService, projectService)
	
	// Create a project and task
	project := domain.NewProject("Test Project", "A test project", "Test planning")
	err := projectService.Create(project)
	require.NoError(t, err)
	
	task := domain.NewTask(project.ID, "Test Task", "A test task")
	err = taskService.Create(task)
	require.NoError(t, err)
	
	// Record a decision
	decision, err := planningService.RecordDecision(
		project.ID,
		"Which database should we use?",
		"PostgreSQL",
		"Better JSON support and performance",
		[]string{"MySQL", "SQLite"},
		true,
		[]string{task.ID},
	)
	
	assert.NoError(t, err)
	assert.NotNil(t, decision)
	assert.Equal(t, "Which database should we use?", decision.Question)
	assert.Equal(t, "PostgreSQL", decision.Choice)
	assert.Equal(t, "Better JSON support and performance", decision.Rationale)
	assert.Contains(t, decision.Alternatives, "MySQL")
	assert.Contains(t, decision.Alternatives, "SQLite")
	assert.True(t, decision.Reversible)
	assert.Contains(t, decision.AffectedTasks, task.ID)
	
	// Verify decision is in the list
	decisions, err := planningService.ListDecisions(project.ID)
	assert.NoError(t, err)
	assert.Len(t, decisions, 1)
	assert.Equal(t, decision.ID, decisions[0].ID)
}

func TestPlanningService_GenerateSessionSummary(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := NewTaskService(memStorage)
	projectService := NewProjectService(memStorage)
	planningService := NewPlanningService(memStorage, taskService, projectService)
	
	// Create a project
	project := domain.NewProject("Test Project", "A test project", "Test planning")
	err := projectService.Create(project)
	require.NoError(t, err)
	
	// Start a planning session
	session, err := planningService.StartPlanningSession(project.ID, "Sprint Planning")
	require.NoError(t, err)
	
	// Create some tasks
	task1 := domain.NewTask(project.ID, "Task 1", "First task")
	task2 := domain.NewTask(project.ID, "Task 2", "Second task")
	err = taskService.Create(task1)
	require.NoError(t, err)
	err = taskService.Create(task2)
	require.NoError(t, err)
	
	// Add tasks to session
	err = planningService.AddTaskToSession(session.ID, task1.ID)
	require.NoError(t, err)
	err = planningService.AddTaskToSession(session.ID, task2.ID)
	require.NoError(t, err)
	
	// Record a discovery and decision
	_, err = planningService.RecordDiscovery(project.ID, "Important insight", domain.ImpactMedium, domain.SourcePlanning, []string{task1.ID})
	require.NoError(t, err)
	_, err = planningService.RecordDecision(project.ID, "Test question", "Test choice", "Test rationale", []string{"alt1"}, true, []string{task2.ID})
	require.NoError(t, err)
	
	// Generate session summary
	summary, err := planningService.GenerateSessionSummary(session.ID)
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, session.ID, summary.Session.ID)
	assert.Len(t, summary.Tasks, 2)
	assert.Len(t, summary.Discoveries, 1)
	assert.Len(t, summary.Decisions, 1)
	assert.Greater(t, summary.Duration, time.Duration(0))
}