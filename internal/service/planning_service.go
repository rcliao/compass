package service

import (
	"fmt"
	"time"

	"github.com/rcliao/compass/internal/domain"
)

type PlanningService struct {
	storage        PlanningStorage
	taskService    *TaskService
	projectService *ProjectService
	headerGen      *HeaderGenerator
}

type PlanningStorage interface {
	CreatePlanningSession(session *domain.PlanningSession) error
	GetPlanningSession(id string) (*domain.PlanningSession, error)
	ListPlanningSessions(projectID string) ([]*domain.PlanningSession, error)
	UpdatePlanningSession(id string, updates map[string]interface{}) (*domain.PlanningSession, error)
	CreateDiscovery(discovery *domain.Discovery) error
	ListDiscoveries(projectID string) ([]*domain.Discovery, error)
	CreateDecision(decision *domain.Decision) error
	ListDecisions(projectID string) ([]*domain.Decision, error)
}

func NewPlanningService(storage PlanningStorage, taskService *TaskService, projectService *ProjectService) *PlanningService {
	return &PlanningService{
		storage:        storage,
		taskService:    taskService,
		projectService: projectService,
		headerGen:      NewHeaderGenerator(200),
	}
}

func (ps *PlanningService) StartPlanningSession(projectID, name string) (*domain.PlanningSession, error) {
	// Verify project exists
	_, err := ps.projectService.Get(projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}
	
	session := domain.NewPlanningSession(projectID, name)
	err = ps.storage.CreatePlanningSession(session)
	if err != nil {
		return nil, err
	}
	
	return session, nil
}

func (ps *PlanningService) GetPlanningSession(id string) (*domain.PlanningSession, error) {
	return ps.storage.GetPlanningSession(id)
}

func (ps *PlanningService) ListPlanningSessions(projectID string) ([]*domain.PlanningSession, error) {
	return ps.storage.ListPlanningSessions(projectID)
}

func (ps *PlanningService) CompletePlanningSession(id string) (*domain.PlanningSession, error) {
	updates := map[string]interface{}{
		"status": domain.PlanningStatusCompleted,
	}
	return ps.storage.UpdatePlanningSession(id, updates)
}

func (ps *PlanningService) AbortPlanningSession(id string) (*domain.PlanningSession, error) {
	updates := map[string]interface{}{
		"status": domain.PlanningStatusAborted,
	}
	return ps.storage.UpdatePlanningSession(id, updates)
}

func (ps *PlanningService) AddTaskToSession(sessionID string, taskID string) error {
	session, err := ps.storage.GetPlanningSession(sessionID)
	if err != nil {
		return err
	}
	
	if session.Status != domain.PlanningStatusActive {
		return fmt.Errorf("cannot add tasks to %s planning session", session.Status)
	}
	
	// Add task to session
	session.Tasks = append(session.Tasks, taskID)
	
	updates := map[string]interface{}{
		"tasks": session.Tasks,
	}
	
	_, err = ps.storage.UpdatePlanningSession(sessionID, updates)
	return err
}

func (ps *PlanningService) RecordDiscovery(projectID, insight string, impact domain.Impact, source domain.DiscoverySource, affectedTaskIDs []string) (*domain.Discovery, error) {
	discovery := domain.NewDiscovery(projectID, insight, impact, source)
	discovery.AffectedTasks = affectedTaskIDs
	
	err := ps.storage.CreateDiscovery(discovery)
	if err != nil {
		return nil, err
	}
	
	// Update affected tasks with discovery reference
	for _, taskID := range affectedTaskIDs {
		task, err := ps.taskService.Get(taskID)
		if err != nil {
			continue // Skip invalid task IDs
		}
		
		// Add discovery ID to task context
		task.Context.Decisions = append(task.Context.Decisions, discovery.ID)
		
		// Update task header to reflect new context
		project, _ := ps.projectService.Get(task.ProjectID)
		ps.headerGen.UpdateTaskHeader(task, project)
		
		updates := map[string]interface{}{
			"decisions":        task.Context.Decisions,
			"contextualHeader": task.Context.ContextualHeader,
			"lastVerified":     task.Context.LastVerified,
		}
		ps.taskService.Update(taskID, updates)
	}
	
	return discovery, nil
}

func (ps *PlanningService) RecordDecision(projectID, question, choice, rationale string, alternatives []string, reversible bool, affectedTaskIDs []string) (*domain.Decision, error) {
	decision := domain.NewDecision(projectID, question, choice, rationale, alternatives, reversible)
	decision.AffectedTasks = affectedTaskIDs
	
	err := ps.storage.CreateDecision(decision)
	if err != nil {
		return nil, err
	}
	
	// Update affected tasks with decision reference
	for _, taskID := range affectedTaskIDs {
		task, err := ps.taskService.Get(taskID)
		if err != nil {
			continue // Skip invalid task IDs
		}
		
		// Add decision ID to task context
		task.Context.Decisions = append(task.Context.Decisions, decision.ID)
		
		// Update task header to reflect new context
		project, _ := ps.projectService.Get(task.ProjectID)
		ps.headerGen.UpdateTaskHeader(task, project)
		
		updates := map[string]interface{}{
			"decisions":        task.Context.Decisions,
			"contextualHeader": task.Context.ContextualHeader,
			"lastVerified":     task.Context.LastVerified,
		}
		ps.taskService.Update(taskID, updates)
	}
	
	return decision, nil
}

func (ps *PlanningService) ListDiscoveries(projectID string) ([]*domain.Discovery, error) {
	return ps.storage.ListDiscoveries(projectID)
}

func (ps *PlanningService) ListDecisions(projectID string) ([]*domain.Decision, error) {
	return ps.storage.ListDecisions(projectID)
}

func (ps *PlanningService) GenerateSessionSummary(sessionID string) (*SessionSummary, error) {
	session, err := ps.storage.GetPlanningSession(sessionID)
	if err != nil {
		return nil, err
	}
	
	// Get session tasks
	var tasks []*domain.Task
	for _, taskID := range session.Tasks {
		if task, err := ps.taskService.Get(taskID); err == nil {
			tasks = append(tasks, task)
		}
	}
	
	// Get discoveries during session timeframe
	discoveries, _ := ps.storage.ListDiscoveries(session.ProjectID)
	var sessionDiscoveries []*domain.Discovery
	for _, discovery := range discoveries {
		if discovery.Timestamp.After(session.CreatedAt) {
			sessionDiscoveries = append(sessionDiscoveries, discovery)
		}
	}
	
	// Get decisions during session timeframe
	decisions, _ := ps.storage.ListDecisions(session.ProjectID)
	var sessionDecisions []*domain.Decision
	for _, decision := range decisions {
		if decision.Timestamp.After(session.CreatedAt) {
			sessionDecisions = append(sessionDecisions, decision)
		}
	}
	
	summary := &SessionSummary{
		Session:     session,
		Tasks:       tasks,
		Discoveries: sessionDiscoveries,
		Decisions:   sessionDecisions,
		Duration:    time.Since(session.CreatedAt),
	}
	
	return summary, nil
}

type SessionSummary struct {
	Session     *domain.PlanningSession `json:"session"`
	Tasks       []*domain.Task          `json:"tasks"`
	Discoveries []*domain.Discovery     `json:"discoveries"`
	Decisions   []*domain.Decision      `json:"decisions"`
	Duration    time.Duration           `json:"duration"`
}