package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rcliao/compass/internal/domain"
)

func TestMemoryStorage_TaskOperations(t *testing.T) {
	storage := NewMemoryStorage()
	
	// Create a test project first
	project := domain.NewProject("Test Project", "A test project", "Test goal")
	err := storage.CreateProject(project)
	require.NoError(t, err)
	
	// Test task creation
	task := domain.NewTask(project.ID, "Test Task", "A test task")
	err = storage.CreateTask(task)
	assert.NoError(t, err)
	
	// Test duplicate creation fails
	err = storage.CreateTask(task)
	assert.Error(t, err)
	
	// Test task retrieval
	retrieved, err := storage.GetTask(task.ID)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)
	assert.Equal(t, task.Card.Title, retrieved.Card.Title)
	
	// Test task listing
	filter := domain.TaskFilter{ProjectID: &project.ID}
	tasks, err := storage.ListTasks(filter)
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task.ID, tasks[0].ID)
	
	// Test task update
	updates := map[string]interface{}{
		"title": "Updated Task Title",
		"status": domain.StatusInProgress,
	}
	updated, err := storage.UpdateTask(task.ID, updates)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Task Title", updated.Card.Title)
	assert.Equal(t, domain.StatusInProgress, updated.Card.Status)
	
	// Test task deletion
	err = storage.DeleteTask(task.ID)
	assert.NoError(t, err)
	
	// Verify task is deleted
	_, err = storage.GetTask(task.ID)
	assert.Error(t, err)
}

func TestMemoryStorage_ProjectOperations(t *testing.T) {
	storage := NewMemoryStorage()
	
	// Test project creation
	project := domain.NewProject("Test Project", "A test project", "Test goal")
	err := storage.CreateProject(project)
	assert.NoError(t, err)
	
	// Test duplicate creation fails
	err = storage.CreateProject(project)
	assert.Error(t, err)
	
	// Test project retrieval
	retrieved, err := storage.GetProject(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, project.ID, retrieved.ID)
	assert.Equal(t, project.Name, retrieved.Name)
	
	// Test project listing
	projects, err := storage.ListProjects()
	assert.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Equal(t, project.ID, projects[0].ID)
	
	// Test setting current project
	err = storage.SetCurrentProject(project.ID)
	assert.NoError(t, err)
	
	// Test getting current project
	current, err := storage.GetCurrentProject()
	assert.NoError(t, err)
	assert.Equal(t, project.ID, current.ID)
	
	// Test setting non-existent project as current fails
	err = storage.SetCurrentProject("non-existent")
	assert.Error(t, err)
}

func TestMemoryStorage_TaskFiltering(t *testing.T) {
	storage := NewMemoryStorage()
	
	// Create test projects
	project1 := domain.NewProject("Project 1", "First project", "Goal 1")
	project2 := domain.NewProject("Project 2", "Second project", "Goal 2")
	storage.CreateProject(project1)
	storage.CreateProject(project2)
	
	// Create tasks in different projects with different statuses
	task1 := domain.NewTask(project1.ID, "Task 1", "Description 1")
	task1.Card.Status = domain.StatusPlanned
	
	task2 := domain.NewTask(project1.ID, "Task 2", "Description 2")
	task2.Card.Status = domain.StatusInProgress
	
	task3 := domain.NewTask(project2.ID, "Task 3", "Description 3")
	task3.Card.Status = domain.StatusPlanned
	
	storage.CreateTask(task1)
	storage.CreateTask(task2)
	storage.CreateTask(task3)
	
	// Test filtering by project
	filter := domain.TaskFilter{ProjectID: &project1.ID}
	tasks, err := storage.ListTasks(filter)
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	
	// Test filtering by status
	status := domain.StatusPlanned
	filter = domain.TaskFilter{Status: &status}
	tasks, err = storage.ListTasks(filter)
	assert.NoError(t, err)
	assert.Len(t, tasks, 2)
	
	// Test filtering by project and status
	filter = domain.TaskFilter{ProjectID: &project1.ID, Status: &status}
	tasks, err = storage.ListTasks(filter)
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
}