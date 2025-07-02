package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/rcliao/compass/internal/domain"
)

type HeaderGenerator struct {
	maxTokens int
}

func NewHeaderGenerator(maxTokens int) *HeaderGenerator {
	if maxTokens <= 0 {
		maxTokens = 200
	}
	return &HeaderGenerator{
		maxTokens: maxTokens,
	}
}

func (g *HeaderGenerator) Generate(task *domain.Task, project *domain.Project) string {
	var parts []string
	
	// Add project context
	if project != nil && project.Goal != "" {
		parts = append(parts, fmt.Sprintf("Part of %s.", project.Goal))
	}
	
	// Add purpose from description
	if task.Card.Description != "" {
		purpose := g.truncate(task.Card.Description, 50)
		parts = append(parts, fmt.Sprintf("Purpose: %s", purpose))
	} else if task.Card.Title != "" {
		// Fallback to title if no description
		parts = append(parts, fmt.Sprintf("Task: %s.", task.Card.Title))
	}
	
	// Add status context
	if task.Card.Status != domain.StatusPlanned {
		parts = append(parts, fmt.Sprintf("Status: %s.", task.Card.Status))
	}
	
	// Add dependencies
	if len(task.Context.Dependencies) > 0 {
		depCount := len(task.Context.Dependencies)
		if depCount <= 3 {
			parts = append(parts, fmt.Sprintf("Depends on: %s.", 
				strings.Join(task.Context.Dependencies, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("Depends on: %s and %d others.", 
				strings.Join(task.Context.Dependencies[:3], ", "), depCount-3))
		}
	}
	
	// Add blockers
	if len(task.Context.Blockers) > 0 {
		blockerText := strings.Join(task.Context.Blockers, ". ")
		parts = append(parts, fmt.Sprintf("Blocked by: %s", g.truncate(blockerText, 60)))
	}
	
	// Add file context
	if len(task.Context.Files) > 0 {
		fileCount := len(task.Context.Files)
		if fileCount <= 3 {
			parts = append(parts, fmt.Sprintf("Affects files: %s.", 
				strings.Join(task.Context.Files, ", ")))
		} else {
			parts = append(parts, fmt.Sprintf("Affects %d files including: %s.", 
				fileCount, strings.Join(task.Context.Files[:3], ", ")))
		}
	}
	
	// Add confidence level if not medium
	if task.Context.Confidence != domain.ConfidenceMedium {
		parts = append(parts, fmt.Sprintf("Confidence: %s.", task.Context.Confidence))
	}
	
	// Add acceptance criteria count
	if len(task.Criteria.Acceptance) > 0 {
		parts = append(parts, fmt.Sprintf("Has %d acceptance criteria.", len(task.Criteria.Acceptance)))
	}
	
	header := strings.Join(parts, " ")
	return g.truncate(header, g.maxTokens)
}

func (g *HeaderGenerator) truncate(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	// Try to truncate at word boundary
	truncated := text[:maxLength]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLength/2 {
		return truncated[:lastSpace] + "..."
	}
	
	return truncated[:maxLength-3] + "..."
}

func (g *HeaderGenerator) UpdateTaskHeader(task *domain.Task, project *domain.Project) {
	header := g.Generate(task, project)
	task.Context.ContextualHeader = header
	task.Context.LastVerified = time.Now()
}

func (g *HeaderGenerator) IsStale(task *domain.Task, maxAge time.Duration) bool {
	if task.Context.LastVerified.IsZero() {
		return true
	}
	
	age := time.Since(task.Context.LastVerified)
	return age > maxAge
}