package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rcliao/compass/internal/domain"
	"github.com/rcliao/compass/internal/storage"
)

func TestHybridSearch_Search(t *testing.T) {
	// Setup storage with test data
	memStorage := storage.NewMemoryStorage()
	
	// Create a project
	project := domain.NewProject("Test Project", "A test project", "Build software")
	err := memStorage.CreateProject(project)
	require.NoError(t, err)
	
	// Create test tasks
	task1 := domain.NewTask(project.ID, "Implement authentication", "Add JWT-based authentication to the API")
	task1.Context.Files = []string{"auth.go", "middleware.go"}
	task1.Context.ContextualHeader = "Part of Build software. Purpose: Add JWT-based authentication to the API. Affects files: auth.go, middleware.go."
	
	task2 := domain.NewTask(project.ID, "Setup database", "Configure PostgreSQL database")
	task2.Context.Files = []string{"database.go", "migrations/"}
	task2.Context.ContextualHeader = "Part of Build software. Purpose: Configure PostgreSQL database. Affects files: database.go, migrations/."
	
	task3 := domain.NewTask(project.ID, "Create user model", "Define user data structures")
	task3.Context.Dependencies = []string{"setup database"}
	task3.Context.ContextualHeader = "Part of Build software. Purpose: Define user data structures. Depends on: Setup database."
	
	err = memStorage.CreateTask(task1)
	require.NoError(t, err)
	err = memStorage.CreateTask(task2)
	require.NoError(t, err)
	err = memStorage.CreateTask(task3)
	require.NoError(t, err)
	
	// Initialize search
	searcher := NewHybridSearch(memStorage)
	
	// Test keyword search
	opts := domain.SearchOptions{ProjectID: &project.ID, Limit: 10}
	results, err := searcher.Search("authentication", opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
	
	// Should find the authentication task
	found := false
	for _, result := range results {
		if result.Task.ID == task1.ID {
			found = true
			assert.Equal(t, "keyword", result.MatchType)
			assert.Greater(t, result.Score, 0.0)
			break
		}
	}
	assert.True(t, found, "Should find authentication task")
	
	// Test structural search
	results, err = searcher.Search("database", opts)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)
	
	// Should find tasks related to database
	foundSetup := false
	foundUser := false
	for _, result := range results {
		if result.Task.ID == task2.ID {
			foundSetup = true
		}
		if result.Task.ID == task3.ID {
			foundUser = true // Should match via dependency search
		}
	}
	assert.True(t, foundSetup, "Should find database setup task")
	assert.True(t, foundUser, "Should find user task via dependency")
}

func TestHybridSearch_KeywordSearch(t *testing.T) {
	searcher := &HybridSearch{}
	
	task := domain.NewTask("project-id", "Implement authentication", "Add JWT-based authentication to the API")
	task.Criteria.Acceptance = []string{"Users can login with JWT tokens"}
	
	// Exact title match should score high
	score := searcher.keywordSearch(task, "implement authentication")
	assert.Greater(t, score, 10.0)
	
	// Partial title match
	score = searcher.keywordSearch(task, "authentication")
	assert.Greater(t, score, 5.0)
	
	// Description match
	score = searcher.keywordSearch(task, "jwt")
	assert.Greater(t, score, 0.0)
	
	// No match
	score = searcher.keywordSearch(task, "database")
	assert.Equal(t, 0.0, score)
}

func TestHybridSearch_StructuralSearch(t *testing.T) {
	searcher := &HybridSearch{}
	
	task := domain.NewTask("project-id", "Test task", "A test task")
	task.Context.Files = []string{"auth.go", "database.go"}
	task.Context.Dependencies = []string{"setup auth service"}
	task.Context.Blockers = []string{"SSL certificate needed"}
	
	// File match
	score := searcher.structuralSearch(task, "auth.go")
	assert.Greater(t, score, 0.0)
	
	// Dependency match (should score higher than file)
	score = searcher.structuralSearch(task, "auth service")
	assert.Greater(t, score, 4.0)
	
	// Blocker match (should score highest)
	score = searcher.structuralSearch(task, "ssl certificate")
	assert.Greater(t, score, 6.0)
	
	// No match
	score = searcher.structuralSearch(task, "nonexistent")
	assert.Equal(t, 0.0, score)
}

func TestHybridSearch_MergeAndRank(t *testing.T) {
	searcher := &HybridSearch{}
	
	task1 := domain.NewTask("project-id", "Task 1", "First task")
	task2 := domain.NewTask("project-id", "Task 2", "Second task")
	
	results := []*domain.SearchResult{
		{Task: task1, Score: 5.0, MatchType: "keyword"},
		{Task: task1, Score: 3.0, MatchType: "structural"}, // Same task, different match
		{Task: task2, Score: 7.0, MatchType: "header"},
	}
	
	merged := searcher.mergeAndRank(results)
	
	// Should have 2 unique tasks
	assert.Len(t, merged, 2)
	
	// Should be sorted by score (task1: 8.0, task2: 7.0)
	assert.Equal(t, task1.ID, merged[0].Task.ID)
	assert.Equal(t, 8.0, merged[0].Score)
	assert.Equal(t, task2.ID, merged[1].Task.ID)
	assert.Equal(t, 7.0, merged[1].Score)
}

func TestHybridSearch_Pagination(t *testing.T) {
	// Setup storage with multiple tasks
	memStorage := storage.NewMemoryStorage()
	
	project := domain.NewProject("Test Project", "A test project", "Build software")
	err := memStorage.CreateProject(project)
	require.NoError(t, err)
	
	// Create 5 tasks that match "test"
	for i := 0; i < 5; i++ {
		task := domain.NewTask(project.ID, "Test task "+string(rune('A'+i)), "A test task")
		err = memStorage.CreateTask(task)
		require.NoError(t, err)
	}
	
	searcher := NewHybridSearch(memStorage)
	
	// Test limit
	opts := domain.SearchOptions{ProjectID: &project.ID, Limit: 3}
	results, err := searcher.Search("test", opts)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
	
	// Test offset
	opts = domain.SearchOptions{ProjectID: &project.ID, Limit: 2, Offset: 2}
	results, err = searcher.Search("test", opts)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}