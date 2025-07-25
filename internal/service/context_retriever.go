package service

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rcliao/compass/internal/domain"
	"github.com/rcliao/compass/internal/search"
)

// TaskCacheEntry represents cached task data
type TaskCacheEntry struct {
	Tasks     []*domain.Task
	Timestamp time.Time
	ProjectID string
}

type ContextRetriever struct {
	taskStorage    TaskStorage
	projectStorage ProjectStorage
	searcher       *search.HybridSearch
	headerGen      *HeaderGenerator
	taskCache      map[string]*TaskCacheEntry
	cacheMu        sync.RWMutex
	cacheTTL       time.Duration
}

func NewContextRetriever(taskStorage TaskStorage, projectStorage ProjectStorage) *ContextRetriever {
	cr := &ContextRetriever{
		taskStorage:    taskStorage,
		projectStorage: projectStorage,
		searcher:       search.NewHybridSearch(taskStorage),
		headerGen:      NewHeaderGenerator(200),
		taskCache:      make(map[string]*TaskCacheEntry),
		cacheTTL:       5 * time.Minute, // 5 minute cache TTL
	}
	
	// Start cache cleanup routine
	go cr.cacheCleanupLoop()
	
	return cr
}

// cacheCleanupLoop periodically cleans up expired cache entries
func (cr *ContextRetriever) cacheCleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		cr.cleanupExpiredCache()
	}
}

// cleanupExpiredCache removes expired entries from the task cache
func (cr *ContextRetriever) cleanupExpiredCache() {
	cr.cacheMu.Lock()
	defer cr.cacheMu.Unlock()
	
	now := time.Now()
	for projectID, entry := range cr.taskCache {
		if now.Sub(entry.Timestamp) > cr.cacheTTL {
			delete(cr.taskCache, projectID)
		}
	}
}

func (cr *ContextRetriever) GetTaskContext(taskID string) (*domain.TaskContext, error) {
	// Get the main task
	task, err := cr.taskStorage.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	
	// Get the project
	project, err := cr.projectStorage.GetProject(task.ProjectID)
	if err != nil {
		return nil, err
	}
	
	// Update header if stale
	if cr.headerGen.IsStale(task, 24*time.Hour) {
		cr.headerGen.UpdateTaskHeader(task, project)
		// Save the updated task
		updates := map[string]interface{}{
			"contextualHeader": task.Context.ContextualHeader,
			"lastVerified":     task.Context.LastVerified,
		}
		cr.taskStorage.UpdateTask(taskID, updates)
	}
	
	// Get dependencies
	dependencies, err := cr.getTaskDependencies(task)
	if err != nil {
		return nil, err
	}
	
	// Get children
	children, err := cr.getTaskChildren(task)
	if err != nil {
		return nil, err
	}
	
	// Get related tasks
	related, err := cr.getRelatedTasks(task, project)
	if err != nil {
		return nil, err
	}
	
	return &domain.TaskContext{
		Task:         task,
		Project:      project,
		Dependencies: dependencies,
		Children:     children,
		Related:      related,
	}, nil
}

func (cr *ContextRetriever) Search(query string, opts domain.SearchOptions) ([]*domain.SearchResult, error) {
	return cr.searcher.Search(query, opts)
}

// getCachedTasks retrieves tasks for a project with caching to reduce file I/O
func (cr *ContextRetriever) getCachedTasks(projectID string) ([]*domain.Task, error) {
	// Check cache first
	cr.cacheMu.RLock()
	if entry, exists := cr.taskCache[projectID]; exists {
		if time.Since(entry.Timestamp) < cr.cacheTTL {
			cr.cacheMu.RUnlock()
			return entry.Tasks, nil
		}
	}
	cr.cacheMu.RUnlock()
	
	// Cache miss or expired, load from storage
	filter := domain.TaskFilter{
		ProjectID: &projectID,
	}
	
	tasks, err := cr.taskStorage.ListTasks(filter)
	if err != nil {
		return nil, err
	}
	
	// Update cache
	cr.cacheMu.Lock()
	cr.taskCache[projectID] = &TaskCacheEntry{
		Tasks:     tasks,
		Timestamp: time.Now(),
		ProjectID: projectID,
	}
	cr.cacheMu.Unlock()
	
	return tasks, nil
}

// InvalidateTaskCache invalidates the task cache for a project (call when tasks are modified)
func (cr *ContextRetriever) InvalidateTaskCache(projectID string) {
	cr.cacheMu.Lock()
	delete(cr.taskCache, projectID)
	cr.cacheMu.Unlock()
}

func (cr *ContextRetriever) GetNextTask(criteria domain.NextTaskCriteria) (*domain.Task, error) {
	// Get all tasks for the project (with caching)
	tasks, err := cr.getCachedTasks(criteria.ProjectID)
	if err != nil {
		return nil, err
	}
	
	// Filter out excluded tasks
	excludeMap := make(map[string]bool)
	for _, id := range criteria.Exclude {
		excludeMap[id] = true
	}
	
	var candidates []*domain.Task
	for _, task := range tasks {
		if excludeMap[task.ID] {
			continue
		}
		
		// Only consider planned or blocked tasks
		if task.Card.Status == domain.StatusPlanned || task.Card.Status == domain.StatusBlocked {
			candidates = append(candidates, task)
		}
	}
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable next task found")
	}
	
	// Score and rank candidates
	scored := cr.scoreTaskCandidates(candidates)
	
	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	
	return scored[0].Task, nil
}

func (cr *ContextRetriever) CheckSufficiency(taskID string) (*domain.SufficiencyReport, error) {
	task, err := cr.taskStorage.GetTask(taskID)
	if err != nil {
		return nil, err
	}
	
	var missing []string
	var stale []string
	
	// Check if task has sufficient context
	if task.Card.Description == "" {
		missing = append(missing, "description")
	}
	
	if len(task.Criteria.Acceptance) == 0 {
		missing = append(missing, "acceptance criteria")
	}
	
	if len(task.Context.Files) == 0 && task.Card.Status != domain.StatusPlanned {
		missing = append(missing, "affected files")
	}
	
	// Check staleness
	if cr.headerGen.IsStale(task, 24*time.Hour) {
		stale = append(stale, "contextual header")
	}
	
	// Check if dependencies are still valid
	for _, depID := range task.Context.Dependencies {
		if _, err := cr.taskStorage.GetTask(depID); err != nil {
			stale = append(stale, fmt.Sprintf("dependency: %s", depID))
		}
	}
	
	sufficient := len(missing) == 0 && len(stale) == 0
	
	return &domain.SufficiencyReport{
		TaskID:     taskID,
		Sufficient: sufficient,
		Missing:    missing,
		Stale:      stale,
	}, nil
}

func (cr *ContextRetriever) getTaskDependencies(task *domain.Task) ([]*domain.Task, error) {
	var dependencies []*domain.Task
	
	for _, depID := range task.Context.Dependencies {
		// Try to parse as task ID first
		if dep, err := cr.taskStorage.GetTask(depID); err == nil {
			dependencies = append(dependencies, dep)
		}
		// If not a task ID, might be a textual dependency - skip for now
	}
	
	return dependencies, nil
}

func (cr *ContextRetriever) getTaskChildren(task *domain.Task) ([]*domain.Task, error) {
	if len(task.Card.Children) == 0 {
		return []*domain.Task{}, nil
	}
	
	var children []*domain.Task
	
	for _, childID := range task.Card.Children {
		if child, err := cr.taskStorage.GetTask(childID); err == nil {
			children = append(children, child)
		}
	}
	
	return children, nil
}

func (cr *ContextRetriever) getRelatedTasks(task *domain.Task, project *domain.Project) ([]*domain.Task, error) {
	// Find tasks that share files or have similar context
	filter := domain.TaskFilter{
		ProjectID: &task.ProjectID,
	}
	
	allTasks, err := cr.taskStorage.ListTasks(filter)
	if err != nil {
		return nil, err
	}
	
	var related []*domain.Task
	
	for _, t := range allTasks {
		if t.ID == task.ID {
			continue
		}
		
		// Check for shared files
		if cr.hasSharedFiles(task, t) {
			related = append(related, t)
			continue
		}
		
		// Check for similar titles/descriptions
		if cr.hasSimilarContent(task, t) {
			related = append(related, t)
			continue
		}
	}
	
	// Limit to top 5 most related
	if len(related) > 5 {
		related = related[:5]
	}
	
	return related, nil
}

func (cr *ContextRetriever) hasSharedFiles(task1, task2 *domain.Task) bool {
	files1 := make(map[string]bool)
	for _, file := range task1.Context.Files {
		files1[strings.ToLower(file)] = true
	}
	
	for _, file := range task2.Context.Files {
		if files1[strings.ToLower(file)] {
			return true
		}
	}
	
	return false
}

func (cr *ContextRetriever) hasSimilarContent(task1, task2 *domain.Task) bool {
	// Simple similarity check based on common words
	words1 := cr.extractWords(task1.Card.Title + " " + task1.Card.Description)
	words2 := cr.extractWords(task2.Card.Title + " " + task2.Card.Description)
	
	commonWords := 0
	for word := range words1 {
		if words2[word] && len(word) > 3 { // Only count significant words
			commonWords++
		}
	}
	
	// If they share 2+ significant words, consider them related
	return commonWords >= 2
}

func (cr *ContextRetriever) extractWords(text string) map[string]bool {
	words := make(map[string]bool)
	text = strings.ToLower(text)
	
	// Simple word extraction - in production, you'd use a proper tokenizer
	fields := strings.Fields(text)
	for _, field := range fields {
		// Remove punctuation and keep only letters
		cleaned := strings.Trim(field, ".,!?;:()")
		if len(cleaned) > 2 {
			words[cleaned] = true
		}
	}
	
	return words
}

type ScoredTask struct {
	Task  *domain.Task
	Score float64
}

func (cr *ContextRetriever) scoreTaskCandidates(tasks []*domain.Task) []ScoredTask {
	var scored []ScoredTask
	
	for _, task := range tasks {
		score := 0.0
		
		// Prefer unblocked tasks
		if task.Card.Status == domain.StatusPlanned {
			score += 10.0
		} else if task.Card.Status == domain.StatusBlocked {
			score += 2.0
		}
		
		// Prefer tasks with fewer dependencies
		depCount := len(task.Context.Dependencies)
		if depCount == 0 {
			score += 5.0
		} else {
			score += 5.0 / float64(depCount+1)
		}
		
		// Prefer high confidence tasks
		switch task.Context.Confidence {
		case domain.ConfidenceHigh:
			score += 3.0
		case domain.ConfidenceMedium:
			score += 1.0
		case domain.ConfidenceLow:
			score += 0.0
		}
		
		// Prefer tasks with clear acceptance criteria
		if len(task.Criteria.Acceptance) > 0 {
			score += 2.0
		}
		
		// Prefer tasks that are not too old (to prevent stagnation)
		age := time.Since(task.Card.CreatedAt)
		if age < 7*24*time.Hour {
			score += 1.0
		}
		
		scored = append(scored, ScoredTask{
			Task:  task,
			Score: score,
		})
	}
	
	return scored
}