package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/rcliao/compass/internal/domain"
)

type ProjectSummaryService struct {
	taskService     *TaskService
	projectService  *ProjectService
	planningService *PlanningService
}

func NewProjectSummaryService(taskService *TaskService, projectService *ProjectService, planningService *PlanningService) *ProjectSummaryService {
	return &ProjectSummaryService{
		taskService:     taskService,
		projectService:  projectService,
		planningService: planningService,
	}
}

type ProjectSummary struct {
	Project         *domain.Project          `json:"project"`
	TaskSummary     *TaskSummary             `json:"taskSummary"`
	Discoveries     []*domain.Discovery      `json:"discoveries"`
	Decisions       []*domain.Decision       `json:"decisions"`
	PlanningSessions []*domain.PlanningSession `json:"planningSessions"`
	Insights        *ProjectInsights         `json:"insights"`
	GeneratedAt     time.Time                `json:"generatedAt"`
}

type TaskSummary struct {
	Total       int                          `json:"total"`
	ByStatus    map[domain.TaskStatus]int    `json:"byStatus"`
	ByConfidence map[domain.Confidence]int   `json:"byConfidence"`
	Recent      []*domain.Task               `json:"recent"`
	Blocked     []*domain.Task               `json:"blocked"`
	Completed   []*domain.Task               `json:"completed"`
}

type ProjectInsights struct {
	VelocityTrend       string                    `json:"velocityTrend"`
	BlockerCount        int                       `json:"blockerCount"`
	HighImpactDiscoveries int                     `json:"highImpactDiscoveries"`
	RecentDecisions     int                       `json:"recentDecisions"`
	ContextHealth       string                    `json:"contextHealth"`
	Recommendations     []string                  `json:"recommendations"`
}

func (pss *ProjectSummaryService) GenerateProjectSummary(projectID string) (*ProjectSummary, error) {
	// Get project
	project, err := pss.projectService.Get(projectID)
	if err != nil {
		return nil, err
	}
	
	// Get all tasks
	filter := domain.TaskFilter{ProjectID: &projectID}
	tasks, err := pss.taskService.List(filter)
	if err != nil {
		return nil, err
	}
	
	// Get discoveries
	discoveries, err := pss.planningService.ListDiscoveries(projectID)
	if err != nil {
		discoveries = []*domain.Discovery{} // Continue with empty list
	}
	
	// Get decisions
	decisions, err := pss.planningService.ListDecisions(projectID)
	if err != nil {
		decisions = []*domain.Decision{} // Continue with empty list
	}
	
	// Get planning sessions
	sessions, err := pss.planningService.ListPlanningSessions(projectID)
	if err != nil {
		sessions = []*domain.PlanningSession{} // Continue with empty list
	}
	
	// Generate task summary
	taskSummary := pss.generateTaskSummary(tasks)
	
	// Generate insights
	insights := pss.generateInsights(tasks, discoveries, decisions, sessions)
	
	return &ProjectSummary{
		Project:          project,
		TaskSummary:      taskSummary,
		Discoveries:      discoveries,
		Decisions:        decisions,
		PlanningSessions: sessions,
		Insights:         insights,
		GeneratedAt:      time.Now(),
	}, nil
}

func (pss *ProjectSummaryService) generateTaskSummary(tasks []*domain.Task) *TaskSummary {
	summary := &TaskSummary{
		Total:        len(tasks),
		ByStatus:     make(map[domain.TaskStatus]int),
		ByConfidence: make(map[domain.Confidence]int),
		Recent:       make([]*domain.Task, 0),
		Blocked:      make([]*domain.Task, 0),
		Completed:    make([]*domain.Task, 0),
	}
	
	// Sort tasks by creation date for recent analysis
	sortedTasks := make([]*domain.Task, len(tasks))
	copy(sortedTasks, tasks)
	sort.Slice(sortedTasks, func(i, j int) bool {
		return sortedTasks[i].Card.CreatedAt.After(sortedTasks[j].Card.CreatedAt)
	})
	
	for _, task := range tasks {
		// Count by status
		summary.ByStatus[task.Card.Status]++
		
		// Count by confidence
		summary.ByConfidence[task.Context.Confidence]++
		
		// Collect blocked tasks
		if task.Card.Status == domain.StatusBlocked {
			summary.Blocked = append(summary.Blocked, task)
		}
		
		// Collect completed tasks
		if task.Card.Status == domain.StatusCompleted {
			summary.Completed = append(summary.Completed, task)
		}
	}
	
	// Get recent tasks (last 5)
	recentCount := 5
	if len(sortedTasks) < recentCount {
		recentCount = len(sortedTasks)
	}
	summary.Recent = sortedTasks[:recentCount]
	
	return summary
}

func (pss *ProjectSummaryService) generateInsights(tasks []*domain.Task, discoveries []*domain.Discovery, decisions []*domain.Decision, sessions []*domain.PlanningSession) *ProjectInsights {
	insights := &ProjectInsights{
		Recommendations: make([]string, 0),
	}
	
	// Analyze velocity trend
	insights.VelocityTrend = pss.analyzeVelocityTrend(tasks)
	
	// Count blockers
	for _, task := range tasks {
		if task.Card.Status == domain.StatusBlocked {
			insights.BlockerCount++
		}
	}
	
	// Count high impact discoveries
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	for _, discovery := range discoveries {
		if discovery.Impact == domain.ImpactHigh {
			insights.HighImpactDiscoveries++
		}
		if discovery.Timestamp.After(weekAgo) {
			insights.RecentDecisions++
		}
	}
	
	// Count recent decisions
	for _, decision := range decisions {
		if decision.Timestamp.After(weekAgo) {
			insights.RecentDecisions++
		}
	}
	
	// Analyze context health
	insights.ContextHealth = pss.analyzeContextHealth(tasks)
	
	// Generate recommendations
	insights.Recommendations = pss.generateRecommendations(tasks, discoveries, decisions, insights)
	
	return insights
}

func (pss *ProjectSummaryService) analyzeVelocityTrend(tasks []*domain.Task) string {
	if len(tasks) == 0 {
		return "no_data"
	}
	
	// Simple analysis based on completed tasks in recent periods
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour)
	
	recentCompleted := 0
	previousCompleted := 0
	
	for _, task := range tasks {
		if task.Card.Status == domain.StatusCompleted {
			if task.Card.UpdatedAt.After(weekAgo) {
				recentCompleted++
			} else if task.Card.UpdatedAt.After(twoWeeksAgo) {
				previousCompleted++
			}
		}
	}
	
	if recentCompleted > previousCompleted {
		return "improving"
	} else if recentCompleted < previousCompleted {
		return "declining"
	} else {
		return "stable"
	}
}

func (pss *ProjectSummaryService) analyzeContextHealth(tasks []*domain.Task) string {
	if len(tasks) == 0 {
		return "good"
	}
	
	totalTasks := len(tasks)
	lowConfidenceTasks := 0
	tasksWithoutAcceptance := 0
	staleTasks := 0
	
	for _, task := range tasks {
		if task.Context.Confidence == domain.ConfidenceLow {
			lowConfidenceTasks++
		}
		
		if len(task.Criteria.Acceptance) == 0 {
			tasksWithoutAcceptance++
		}
		
		// Check if context is stale (older than 7 days)
		if time.Since(task.Context.LastVerified) > 7*24*time.Hour {
			staleTasks++
		}
	}
	
	// Calculate health score
	healthScore := 100
	healthScore -= (lowConfidenceTasks * 100) / totalTasks / 3
	healthScore -= (tasksWithoutAcceptance * 100) / totalTasks / 3
	healthScore -= (staleTasks * 100) / totalTasks / 3
	
	if healthScore >= 80 {
		return "excellent"
	} else if healthScore >= 60 {
		return "good"
	} else if healthScore >= 40 {
		return "fair"
	} else {
		return "poor"
	}
}

func (pss *ProjectSummaryService) generateRecommendations(tasks []*domain.Task, discoveries []*domain.Discovery, decisions []*domain.Decision, insights *ProjectInsights) []string {
	recommendations := make([]string, 0)
	
	// Blocker recommendations
	if insights.BlockerCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Address %d blocked tasks to improve velocity", insights.BlockerCount))
	}
	
	// Context health recommendations
	switch insights.ContextHealth {
	case "poor":
		recommendations = append(recommendations, "Review and update task context - many tasks have poor context health")
	case "fair":
		recommendations = append(recommendations, "Consider updating acceptance criteria and task confidence levels")
	}
	
	// Velocity recommendations
	if insights.VelocityTrend == "declining" {
		recommendations = append(recommendations, "Velocity is declining - consider breaking down large tasks or addressing blockers")
	}
	
	// Discovery recommendations
	if insights.HighImpactDiscoveries > 0 {
		recommendations = append(recommendations, "Review high-impact discoveries and update related tasks accordingly")
	}
	
	// Planning recommendations
	lowConfidenceTasks := 0
	for _, task := range tasks {
		if task.Context.Confidence == domain.ConfidenceLow && task.Card.Status == domain.StatusPlanned {
			lowConfidenceTasks++
		}
	}
	
	if lowConfidenceTasks > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Plan session recommended - %d tasks have low confidence", lowConfidenceTasks))
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Project is on track - consider starting a new planning session for next iteration")
	}
	
	return recommendations
}