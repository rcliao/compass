package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rcliao/compass/internal/service"
	"github.com/rcliao/compass/internal/storage"
)

func TestMCPServer_ProjectCommands(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := service.NewTaskService(memStorage)
	projectService := service.NewProjectService(memStorage)
	contextRetriever := service.NewContextRetriever(memStorage, memStorage)
	server := NewMCPServer(taskService, projectService, contextRetriever)

	// Test project creation
	createParams := CreateProjectParams{
		Name:        "Test Project",
		Description: "A test project",
		Goal:        "Test MCP integration",
	}
	createParamsJSON, err := json.Marshal(createParams)
	require.NoError(t, err)

	result, err := server.HandleCommand("compass.project.create", createParamsJSON)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test project listing
	result, err = server.HandleCommand("compass.project.list", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Just verify we got a non-nil result - detailed type checking would require more complex assertions
	assert.NotNil(t, result)
}

func TestMCPServer_TaskCommands(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := service.NewTaskService(memStorage)
	projectService := service.NewProjectService(memStorage)
	contextRetriever := service.NewContextRetriever(memStorage, memStorage)
	server := NewMCPServer(taskService, projectService, contextRetriever)

	// Create a project first
	createProjectParams := CreateProjectParams{
		Name:        "Test Project",
		Description: "A test project",
		Goal:        "Test task operations",
	}
	createProjectParamsJSON, err := json.Marshal(createProjectParams)
	require.NoError(t, err)

	projectResult, err := server.HandleCommand("compass.project.create", createProjectParamsJSON)
	require.NoError(t, err)
	require.NotNil(t, projectResult)

	// Extract project ID (this would need proper type assertion in real code)
	projectData, err := json.Marshal(projectResult)
	require.NoError(t, err)
	
	var project map[string]interface{}
	err = json.Unmarshal(projectData, &project)
	require.NoError(t, err)
	
	projectID, ok := project["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, projectID)

	// Test task creation
	createTaskParams := CreateTaskParams{
		ProjectID:   projectID,
		Title:       "Test Task",
		Description: "A test task",
		Files:       []string{"test.go"},
		Acceptance:  []string{"Task should work"},
	}
	createTaskParamsJSON, err := json.Marshal(createTaskParams)
	require.NoError(t, err)

	taskResult, err := server.HandleCommand("compass.task.create", createTaskParamsJSON)
	assert.NoError(t, err)
	assert.NotNil(t, taskResult)

	// Test task listing
	listParams := ListTasksParams{ProjectID: &projectID}
	listParamsJSON, err := json.Marshal(listParams)
	require.NoError(t, err)

	result, err := server.HandleCommand("compass.task.list", listParamsJSON)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestMCPServer_UnknownCommand(t *testing.T) {
	// Setup
	memStorage := storage.NewMemoryStorage()
	taskService := service.NewTaskService(memStorage)
	projectService := service.NewProjectService(memStorage)
	contextRetriever := service.NewContextRetriever(memStorage, memStorage)
	server := NewMCPServer(taskService, projectService, contextRetriever)

	// Test unknown command
	result, err := server.HandleCommand("compass.unknown.command", nil)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unknown method")
}