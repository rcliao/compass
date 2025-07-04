package mcp

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/rcliao/compass/internal/domain"
	"github.com/rcliao/compass/internal/service"
)

type MCPServer struct {
	taskService       *service.TaskService
	projectService    *service.ProjectService
	contextRetriever  *service.ContextRetriever
	planningService   *service.PlanningService
	summaryService    *service.ProjectSummaryService
}

func NewMCPServer(taskService *service.TaskService, projectService *service.ProjectService, contextRetriever *service.ContextRetriever, planningService *service.PlanningService, summaryService *service.ProjectSummaryService) *MCPServer {
	return &MCPServer{
		taskService:      taskService,
		projectService:   projectService,
		contextRetriever: contextRetriever,
		planningService:  planningService,
		summaryService:   summaryService,
	}
}

type MCPRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *MCPServer) HandleCommand(method string, params json.RawMessage) (interface{}, error) {
	log.Printf("Handling MCP command: %s", method)
	
	switch method {
	// Project commands
	case "compass.project.create":
		return s.handleProjectCreate(params)
	case "compass.project.list":
		return s.handleProjectList()
	case "compass.project.current":
		return s.handleProjectCurrent()
	case "compass.project.set_current":
		return s.handleProjectSetCurrent(params)
		
	// Task commands
	case "compass.task.create":
		return s.handleTaskCreate(params)
	case "compass.task.update":
		return s.handleTaskUpdate(params)
	case "compass.task.list":
		return s.handleTaskList(params)
	case "compass.task.get":
		return s.handleTaskGet(params)
	case "compass.task.delete":
		return s.handleTaskDelete(params)
		
	// Context commands
	case "compass.context.get":
		return s.handleContextGet(params)
	case "compass.context.search":
		return s.handleContextSearch(params)
	case "compass.context.check":
		return s.handleContextCheck(params)
		
	// Intelligent queries
	case "compass.next":
		return s.handleGetNextTask(params)
	case "compass.blockers":
		return s.handleGetBlockers(params)
		
	// Planning commands
	case "compass.planning.start":
		return s.handlePlanningStart(params)
	case "compass.planning.list":
		return s.handlePlanningList(params)
	case "compass.planning.get":
		return s.handlePlanningGet(params)
	case "compass.planning.complete":
		return s.handlePlanningComplete(params)
	case "compass.planning.abort":
		return s.handlePlanningAbort(params)
	case "compass.discovery.add":
		return s.handleDiscoveryAdd(params)
	case "compass.discovery.list":
		return s.handleDiscoveryList(params)
	case "compass.decision.record":
		return s.handleDecisionRecord(params)
	case "compass.decision.list":
		return s.handleDecisionList(params)
		
	// Summary commands
	case "compass.project.summary":
		return s.handleProjectSummary(params)
		
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

// Project handlers
type CreateProjectParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Goal        string `json:"goal"`
}

func (s *MCPServer) handleProjectCreate(params json.RawMessage) (interface{}, error) {
	var p CreateProjectParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	project := domain.NewProject(p.Name, p.Description, p.Goal)
	if err := s.projectService.Create(project); err != nil {
		return nil, err
	}
	
	return project, nil
}

func (s *MCPServer) handleProjectList() (interface{}, error) {
	return s.projectService.List()
}

func (s *MCPServer) handleProjectCurrent() (interface{}, error) {
	return s.projectService.GetCurrent()
}

type SetCurrentProjectParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handleProjectSetCurrent(params json.RawMessage) (interface{}, error) {
	var p SetCurrentProjectParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if err := s.projectService.SetCurrent(p.ID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "success"}, nil
}

// Task handlers
type CreateTaskParams struct {
	ProjectID   string   `json:"projectId"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Files       []string `json:"files,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Acceptance  []string `json:"acceptance,omitempty"`
}

func (s *MCPServer) handleTaskCreate(params json.RawMessage) (interface{}, error) {
	var p CreateTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	task := domain.NewTask(p.ProjectID, p.Title, p.Description)
	
	// Apply optional fields
	if len(p.Files) > 0 {
		task.Context.Files = p.Files
	}
	if len(p.Dependencies) > 0 {
		task.Context.Dependencies = p.Dependencies
	}
	if len(p.Acceptance) > 0 {
		task.Criteria.Acceptance = p.Acceptance
	}
	
	if err := s.taskService.Create(task); err != nil {
		return nil, err
	}
	
	return task, nil
}

type UpdateTaskParams struct {
	ID      string                 `json:"id"`
	Updates map[string]interface{} `json:"updates"`
}

func (s *MCPServer) handleTaskUpdate(params json.RawMessage) (interface{}, error) {
	var p UpdateTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.taskService.Update(p.ID, p.Updates)
}

type ListTasksParams struct {
	ProjectID *string             `json:"projectId,omitempty"`
	Status    *domain.TaskStatus  `json:"status,omitempty"`
	Parent    *string             `json:"parent,omitempty"`
}

func (s *MCPServer) handleTaskList(params json.RawMessage) (interface{}, error) {
	var p ListTasksParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	filter := domain.TaskFilter{
		ProjectID: p.ProjectID,
		Status:    p.Status,
		Parent:    p.Parent,
	}
	
	return s.taskService.List(filter)
}

type GetTaskParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handleTaskGet(params json.RawMessage) (interface{}, error) {
	var p GetTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.taskService.Get(p.ID)
}

type DeleteTaskParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handleTaskDelete(params json.RawMessage) (interface{}, error) {
	var p DeleteTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if err := s.taskService.Delete(p.ID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "success"}, nil
}

// Context handlers
type GetContextParams struct {
	TaskID string `json:"taskId"`
}

func (s *MCPServer) handleContextGet(params json.RawMessage) (interface{}, error) {
	var p GetContextParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.contextRetriever.GetTaskContext(p.TaskID)
}

type SearchContextParams struct {
	Query     string  `json:"query"`
	ProjectID *string `json:"projectId,omitempty"`
	Limit     int     `json:"limit,omitempty"`
	Offset    int     `json:"offset,omitempty"`
}

func (s *MCPServer) handleContextSearch(params json.RawMessage) (interface{}, error) {
	var p SearchContextParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	opts := domain.SearchOptions{
		ProjectID: p.ProjectID,
		Limit:     p.Limit,
		Offset:    p.Offset,
	}
	
	if opts.Limit == 0 {
		opts.Limit = 10
	}
	
	return s.contextRetriever.Search(p.Query, opts)
}

type CheckContextParams struct {
	TaskID string `json:"taskId"`
}

func (s *MCPServer) handleContextCheck(params json.RawMessage) (interface{}, error) {
	var p CheckContextParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.contextRetriever.CheckSufficiency(p.TaskID)
}

type GetNextTaskParams struct {
	ProjectID string   `json:"projectId,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
}

func (s *MCPServer) handleGetNextTask(params json.RawMessage) (interface{}, error) {
	var p GetNextTaskParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	criteria := domain.NextTaskCriteria{
		ProjectID: projectID,
		Exclude:   p.Exclude,
	}
	
	return s.contextRetriever.GetNextTask(criteria)
}

type GetBlockersParams struct {
	ProjectID string `json:"projectId,omitempty"`
}

func (s *MCPServer) handleGetBlockers(params json.RawMessage) (interface{}, error) {
	var p GetBlockersParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	// Get all blocked tasks
	status := domain.StatusBlocked
	filter := domain.TaskFilter{
		ProjectID: &projectID,
		Status:    &status,
	}
	
	return s.taskService.List(filter)
}

// Planning handlers
type StartPlanningParams struct {
	ProjectID string `json:"projectId,omitempty"`
	Name      string `json:"name"`
}

func (s *MCPServer) handlePlanningStart(params json.RawMessage) (interface{}, error) {
	var p StartPlanningParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	return s.planningService.StartPlanningSession(projectID, p.Name)
}

type ListPlanningParams struct {
	ProjectID string `json:"projectId,omitempty"`
}

func (s *MCPServer) handlePlanningList(params json.RawMessage) (interface{}, error) {
	var p ListPlanningParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	return s.planningService.ListPlanningSessions(projectID)
}

type GetPlanningParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handlePlanningGet(params json.RawMessage) (interface{}, error) {
	var p GetPlanningParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.planningService.GetPlanningSession(p.ID)
}

type CompletePlanningParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handlePlanningComplete(params json.RawMessage) (interface{}, error) {
	var p CompletePlanningParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.planningService.CompletePlanningSession(p.ID)
}

type AbortPlanningParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handlePlanningAbort(params json.RawMessage) (interface{}, error) {
	var p AbortPlanningParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.planningService.AbortPlanningSession(p.ID)
}

type AddDiscoveryParams struct {
	ProjectID        string                  `json:"projectId,omitempty"`
	Insight          string                  `json:"insight"`
	Impact           domain.Impact           `json:"impact"`
	Source           domain.DiscoverySource  `json:"source"`
	AffectedTaskIDs  []string               `json:"affectedTaskIds,omitempty"`
}

func (s *MCPServer) handleDiscoveryAdd(params json.RawMessage) (interface{}, error) {
	var p AddDiscoveryParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	return s.planningService.RecordDiscovery(projectID, p.Insight, p.Impact, p.Source, p.AffectedTaskIDs)
}

type ListDiscoveryParams struct {
	ProjectID string `json:"projectId,omitempty"`
}

func (s *MCPServer) handleDiscoveryList(params json.RawMessage) (interface{}, error) {
	var p ListDiscoveryParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	return s.planningService.ListDiscoveries(projectID)
}

type RecordDecisionParams struct {
	ProjectID        string   `json:"projectId,omitempty"`
	Question         string   `json:"question"`
	Choice           string   `json:"choice"`
	Rationale        string   `json:"rationale"`
	Alternatives     []string `json:"alternatives,omitempty"`
	Reversible       bool     `json:"reversible"`
	AffectedTaskIDs  []string `json:"affectedTaskIds,omitempty"`
}

func (s *MCPServer) handleDecisionRecord(params json.RawMessage) (interface{}, error) {
	var p RecordDecisionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	return s.planningService.RecordDecision(projectID, p.Question, p.Choice, p.Rationale, p.Alternatives, p.Reversible, p.AffectedTaskIDs)
}

type ListDecisionParams struct {
	ProjectID string `json:"projectId,omitempty"`
}

func (s *MCPServer) handleDecisionList(params json.RawMessage) (interface{}, error) {
	var p ListDecisionParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	return s.planningService.ListDecisions(projectID)
}

type ProjectSummaryParams struct {
	ProjectID string `json:"projectId,omitempty"`
}

func (s *MCPServer) handleProjectSummary(params json.RawMessage) (interface{}, error) {
	var p ProjectSummaryParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == "" {
		current, err := s.projectService.GetCurrent()
		if err != nil {
			return nil, fmt.Errorf("no current project set and no projectId provided")
		}
		projectID = current.ID
	}
	
	return s.summaryService.GenerateProjectSummary(projectID)
}