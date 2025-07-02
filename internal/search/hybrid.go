package search

import (
	"sort"
	"strings"

	"github.com/rcliao/compass/internal/domain"
)

type HybridSearch struct {
	storage TaskStorage
}

type TaskStorage interface {
	ListTasks(filter domain.TaskFilter) ([]*domain.Task, error)
}

func NewHybridSearch(storage TaskStorage) *HybridSearch {
	return &HybridSearch{
		storage: storage,
	}
}

func (hs *HybridSearch) Search(query string, opts domain.SearchOptions) ([]*domain.SearchResult, error) {
	// Get all tasks for the project or all projects
	filter := domain.TaskFilter{}
	if opts.ProjectID != nil {
		filter.ProjectID = opts.ProjectID
	}
	
	tasks, err := hs.storage.ListTasks(filter)
	if err != nil {
		return nil, err
	}
	
	var results []*domain.SearchResult
	queryLower := strings.ToLower(query)
	
	for _, task := range tasks {
		// Strategy 1: Keyword search in titles/descriptions
		keywordScore := hs.keywordSearch(task, queryLower)
		if keywordScore > 0 {
			results = append(results, &domain.SearchResult{
				Task:      task,
				Score:     keywordScore,
				MatchType: "keyword",
				Snippet:   hs.generateKeywordSnippet(task, queryLower),
			})
		}
		
		// Strategy 2: Contextual header search
		headerScore := hs.headerSearch(task, queryLower)
		if headerScore > 0 {
			results = append(results, &domain.SearchResult{
				Task:      task,
				Score:     headerScore,
				MatchType: "header",
				Snippet:   hs.generateHeaderSnippet(task, queryLower),
			})
		}
		
		// Strategy 3: Structural search (dependencies/blockers/files)
		structuralScore := hs.structuralSearch(task, queryLower)
		if structuralScore > 0 {
			results = append(results, &domain.SearchResult{
				Task:      task,
				Score:     structuralScore,
				MatchType: "structural",
				Snippet:   hs.generateStructuralSnippet(task, queryLower),
			})
		}
	}
	
	// Merge and rank results
	merged := hs.mergeAndRank(results)
	
	// Apply pagination
	if opts.Limit > 0 {
		end := opts.Offset + opts.Limit
		if end > len(merged) {
			end = len(merged)
		}
		if opts.Offset < len(merged) {
			merged = merged[opts.Offset:end]
		} else {
			merged = []*domain.SearchResult{}
		}
	}
	
	return merged, nil
}

func (hs *HybridSearch) keywordSearch(task *domain.Task, query string) float64 {
	score := 0.0
	
	// Title matches are weighted heavily
	titleLower := strings.ToLower(task.Card.Title)
	if strings.Contains(titleLower, query) {
		score += 10.0
		if titleLower == query {
			score += 5.0 // Exact match bonus
		}
	}
	
	// Description matches
	descLower := strings.ToLower(task.Card.Description)
	if strings.Contains(descLower, query) {
		score += 5.0
	}
	
	// Acceptance criteria matches
	for _, criteria := range task.Criteria.Acceptance {
		if strings.Contains(strings.ToLower(criteria), query) {
			score += 3.0
		}
	}
	
	return score
}

func (hs *HybridSearch) headerSearch(task *domain.Task, query string) float64 {
	if task.Context.ContextualHeader == "" {
		return 0.0
	}
	
	headerLower := strings.ToLower(task.Context.ContextualHeader)
	if strings.Contains(headerLower, query) {
		return 7.0
	}
	
	return 0.0
}

func (hs *HybridSearch) structuralSearch(task *domain.Task, query string) float64 {
	score := 0.0
	
	// File matches
	for _, file := range task.Context.Files {
		if strings.Contains(strings.ToLower(file), query) {
			score += 4.0
		}
	}
	
	// Dependency matches
	for _, dep := range task.Context.Dependencies {
		if strings.Contains(strings.ToLower(dep), query) {
			score += 6.0 // Dependencies are important
		}
	}
	
	// Blocker matches
	for _, blocker := range task.Context.Blockers {
		if strings.Contains(strings.ToLower(blocker), query) {
			score += 8.0 // Blockers are very important
		}
	}
	
	return score
}

func (hs *HybridSearch) generateKeywordSnippet(task *domain.Task, query string) string {
	// Try title first
	titleLower := strings.ToLower(task.Card.Title)
	if strings.Contains(titleLower, query) {
		return hs.highlightText(task.Card.Title, query)
	}
	
	// Then description
	descLower := strings.ToLower(task.Card.Description)
	if strings.Contains(descLower, query) {
		return hs.extractSnippet(task.Card.Description, query, 100)
	}
	
	return task.Card.Title
}

func (hs *HybridSearch) generateHeaderSnippet(task *domain.Task, query string) string {
	if task.Context.ContextualHeader == "" {
		return task.Card.Title
	}
	
	return hs.extractSnippet(task.Context.ContextualHeader, query, 100)
}

func (hs *HybridSearch) generateStructuralSnippet(task *domain.Task, query string) string {
	// Check files
	for _, file := range task.Context.Files {
		if strings.Contains(strings.ToLower(file), query) {
			return "File: " + hs.highlightText(file, query)
		}
	}
	
	// Check dependencies
	for _, dep := range task.Context.Dependencies {
		if strings.Contains(strings.ToLower(dep), query) {
			return "Dependency: " + hs.highlightText(dep, query)
		}
	}
	
	// Check blockers
	for _, blocker := range task.Context.Blockers {
		if strings.Contains(strings.ToLower(blocker), query) {
			return "Blocker: " + hs.highlightText(blocker, query)
		}
	}
	
	return task.Card.Title
}

func (hs *HybridSearch) extractSnippet(text, query string, maxLength int) string {
	textLower := strings.ToLower(text)
	queryLower := strings.ToLower(query)
	
	index := strings.Index(textLower, queryLower)
	if index == -1 {
		if len(text) > maxLength {
			return text[:maxLength] + "..."
		}
		return text
	}
	
	// Extract context around the match
	start := index - 30
	if start < 0 {
		start = 0
	}
	
	end := index + len(query) + 30
	if end > len(text) {
		end = len(text)
	}
	
	snippet := text[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}
	
	return hs.highlightText(snippet, query)
}

func (hs *HybridSearch) highlightText(text, query string) string {
	// Simple highlighting - in a real implementation you might use different markers
	textLower := strings.ToLower(text)
	queryLower := strings.ToLower(query)
	
	index := strings.Index(textLower, queryLower)
	if index == -1 {
		return text
	}
	
	before := text[:index]
	match := text[index : index+len(query)]
	after := text[index+len(query):]
	
	return before + "**" + match + "**" + after
}

func (hs *HybridSearch) mergeAndRank(results []*domain.SearchResult) []*domain.SearchResult {
	// Group by task ID and sum scores
	taskScores := make(map[string]*domain.SearchResult)
	
	for _, result := range results {
		existing, exists := taskScores[result.Task.ID]
		if exists {
			// Combine scores and choose best snippet
			existing.Score += result.Score
			if result.Score > existing.Score-result.Score { // If this result had higher individual score
				existing.MatchType = result.MatchType
				existing.Snippet = result.Snippet
			}
		} else {
			taskScores[result.Task.ID] = result
		}
	}
	
	// Convert back to slice
	var merged []*domain.SearchResult
	for _, result := range taskScores {
		merged = append(merged, result)
	}
	
	// Sort by score descending
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})
	
	return merged
}