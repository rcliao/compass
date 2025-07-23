package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/rcliao/compass/internal/domain"
	"github.com/rcliao/compass/internal/service"
)

type MCPServer struct {
	taskService       *service.TaskService
	projectService    *service.ProjectService
	contextRetriever  *service.ContextRetriever
	planningService   *service.PlanningService
	summaryService    *service.ProjectSummaryService
	processService    *service.ProcessService
}

func NewMCPServer(taskService *service.TaskService, projectService *service.ProjectService, contextRetriever *service.ContextRetriever, planningService *service.PlanningService, summaryService *service.ProjectSummaryService, processService *service.ProcessService) *MCPServer {
	return &MCPServer{
		taskService:      taskService,
		projectService:   projectService,
		contextRetriever: contextRetriever,
		planningService:  planningService,
		summaryService:   summaryService,
		processService:   processService,
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
		
	// Process commands
	case "compass.process.create":
		return s.handleProcessCreate(params)
	case "compass.process.start":
		return s.handleProcessStart(params)
	case "compass.process.stop":
		return s.handleProcessStop(params)
	case "compass.process.list":
		return s.handleProcessList(params)
	case "compass.process.get":
		return s.handleProcessGet(params)
	case "compass.process.logs":
		return s.handleProcessLogs(params)
	case "compass.process.status":
		return s.handleProcessStatus(params)
	case "compass.process.update":
		return s.handleProcessUpdate(params)
	case "compass.process.group.create":
		return s.handleProcessGroupCreate(params)
	case "compass.process.group.start":
		return s.handleProcessGroupStart(params)
	case "compass.process.group.stop":
		return s.handleProcessGroupStop(params)
		
	// TODO commands
	case "compass.todo.create":
		return s.handleTodoCreate(params)
	case "compass.todo.quick":
		return s.handleTodoQuickCreate(params)
	case "compass.todo.complete":
		return s.handleTodoComplete(params)
	case "compass.todo.reopen":
		return s.handleTodoReopen(params)
	case "compass.todo.list":
		return s.handleTodoList(params)
	case "compass.todo.overdue":
		return s.handleTodoOverdue(params)
	case "compass.todo.priority":
		return s.handleTodoUpdatePriority(params)
	case "compass.todo.due":
		return s.handleTodoSetDue(params)
	case "compass.todo.label.add":
		return s.handleTodoAddLabel(params)
	case "compass.todo.label.remove":
		return s.handleTodoRemoveLabel(params)
	case "compass.todo.progress":
		return s.handleTodoUpdateProgress(params)
		
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
	projects, err := s.projectService.List()
	if err != nil {
		return nil, err
	}
	
	// Return markdown formatted string
	return FormatProjectsAsMarkdown(projects), nil
}

func (s *MCPServer) handleProjectCurrent() (interface{}, error) {
	project, err := s.projectService.GetCurrent()
	if err != nil {
		return nil, err
	}
	
	// Return markdown formatted string for single project
	return fmt.Sprintf("## 📍 Current Project: %s\n\n**ID:** `%s`\n**Description:** %s\n**Goal:** %s", 
		project.Name, project.ID[:8], project.Description, project.Goal), nil
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

// Process handlers
type CreateProcessParams struct {
	ProjectID   string            `json:"projectId,omitempty"`
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	WorkingDir  string            `json:"workingDir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Type        domain.ProcessType `json:"type,omitempty"`
	Port        int               `json:"port,omitempty"`
	Template    string            `json:"template,omitempty"`
}

func (s *MCPServer) handleProcessCreate(params json.RawMessage) (interface{}, error) {
	var p CreateProcessParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Handle template processing
	if p.Template != "" {
		if err := s.applyProcessTemplate(&p); err != nil {
			return nil, fmt.Errorf("failed to apply template: %w", err)
		}
	}
	
	// Validate required parameters
	if p.Name == "" {
		return nil, fmt.Errorf("process name is required")
	}
	if p.Command == "" {
		return nil, fmt.Errorf("command is required")
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
	
	process := domain.NewProcess(projectID, p.Name, p.Command, p.Args)
	
	// Apply optional fields
	if p.WorkingDir != "" {
		process.WorkingDir = p.WorkingDir
	}
	if len(p.Environment) > 0 {
		process.Environment = p.Environment
	}
	if p.Type != "" {
		// Validate process type
		validTypes := []domain.ProcessType{
			domain.ProcessTypeWebServer, domain.ProcessTypeAPIServer, domain.ProcessTypeBuildTool,
			domain.ProcessTypeWatcher, domain.ProcessTypeTest, domain.ProcessTypeDatabase, domain.ProcessTypeCustom,
		}
		isValid := false
		for _, t := range validTypes {
			if p.Type == t {
				isValid = true
				break
			}
		}
		if !isValid {
			validTypeStrings := make([]string, len(validTypes))
			for i, t := range validTypes {
				validTypeStrings[i] = string(t)
			}
			return nil, fmt.Errorf("invalid process type: %s. Must be one of: %v", p.Type, validTypeStrings)
		}
		process.Type = p.Type
	}
	if p.Port > 0 {
		if p.Port > 65535 {
			return nil, fmt.Errorf("invalid port number: %d. Must be between 1 and 65535", p.Port)
		}
		process.Port = p.Port
	}
	
	if err := s.processService.Create(process); err != nil {
		return nil, err
	}
	
	return process, nil
}

// applyProcessTemplate applies a predefined template to process parameters
func (s *MCPServer) applyProcessTemplate(params *CreateProcessParams) error {
	templates := map[string]func(*CreateProcessParams){
		"react-dev": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "npm" }
			if len(p.Args) == 0 { p.Args = []string{"run", "dev"} }
			if p.Type == "" { p.Type = domain.ProcessTypeWebServer }
			if p.Port == 0 { p.Port = 3000 }
			if p.Name == "" { p.Name = "React Development Server" }
		},
		"next-dev": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "npm" }
			if len(p.Args) == 0 { p.Args = []string{"run", "dev"} }
			if p.Type == "" { p.Type = domain.ProcessTypeWebServer }
			if p.Port == 0 { p.Port = 3000 }
			if p.Name == "" { p.Name = "Next.js Development Server" }
		},
		"vite-dev": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "npm" }
			if len(p.Args) == 0 { p.Args = []string{"run", "dev"} }
			if p.Type == "" { p.Type = domain.ProcessTypeWebServer }
			if p.Port == 0 { p.Port = 5173 }
			if p.Name == "" { p.Name = "Vite Development Server" }
		},
		"node-server": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "node" }
			if len(p.Args) == 0 { p.Args = []string{"server.js"} }
			if p.Type == "" { p.Type = domain.ProcessTypeAPIServer }
			if p.Port == 0 { p.Port = 8000 }
			if p.Name == "" { p.Name = "Node.js Server" }
		},
		"express-dev": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "npm" }
			if len(p.Args) == 0 { p.Args = []string{"run", "dev"} }
			if p.Type == "" { p.Type = domain.ProcessTypeAPIServer }
			if p.Port == 0 { p.Port = 3001 }
			if p.Name == "" { p.Name = "Express Development Server" }
		},
		"python-server": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "python" }
			if len(p.Args) == 0 { p.Args = []string{"-m", "http.server"} }
			if p.Type == "" { p.Type = domain.ProcessTypeWebServer }
			if p.Port == 0 { p.Port = 8000 }
			if p.Name == "" { p.Name = "Python HTTP Server" }
		},
		"flask-dev": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "flask" }
			if len(p.Args) == 0 { p.Args = []string{"run", "--debug"} }
			if p.Type == "" { p.Type = domain.ProcessTypeAPIServer }
			if p.Port == 0 { p.Port = 5000 }
			if p.Name == "" { p.Name = "Flask Development Server" }
			if p.Environment == nil { p.Environment = make(map[string]string) }
			if p.Environment["FLASK_ENV"] == "" { p.Environment["FLASK_ENV"] = "development" }
		},
		"django-dev": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "python" }
			if len(p.Args) == 0 { p.Args = []string{"manage.py", "runserver"} }
			if p.Type == "" { p.Type = domain.ProcessTypeWebServer }
			if p.Port == 0 { p.Port = 8000 }
			if p.Name == "" { p.Name = "Django Development Server" }
		},
		"go-server": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "go" }
			if len(p.Args) == 0 { p.Args = []string{"run", "main.go"} }
			if p.Type == "" { p.Type = domain.ProcessTypeAPIServer }
			if p.Port == 0 { p.Port = 8080 }
			if p.Name == "" { p.Name = "Go Server" }
		},
		"webpack-dev": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "npx" }
			if len(p.Args) == 0 { p.Args = []string{"webpack", "serve", "--mode", "development"} }
			if p.Type == "" { p.Type = domain.ProcessTypeBuildTool }
			if p.Port == 0 { p.Port = 8080 }
			if p.Name == "" { p.Name = "Webpack Dev Server" }
		},
		"tailwind-watch": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "npx" }
			if len(p.Args) == 0 { p.Args = []string{"tailwindcss", "-i", "./src/input.css", "-o", "./dist/output.css", "--watch"} }
			if p.Type == "" { p.Type = domain.ProcessTypeWatcher }
			if p.Name == "" { p.Name = "Tailwind CSS Watcher" }
		},
		"postgres": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "postgres" }
			if p.Type == "" { p.Type = domain.ProcessTypeDatabase }
			if p.Port == 0 { p.Port = 5432 }
			if p.Name == "" { p.Name = "PostgreSQL Database" }
		},
		"redis": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "redis-server" }
			if p.Type == "" { p.Type = domain.ProcessTypeDatabase }
			if p.Port == 0 { p.Port = 6379 }
			if p.Name == "" { p.Name = "Redis Server" }
		},
		"mysql": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "mysqld" }
			if p.Type == "" { p.Type = domain.ProcessTypeDatabase }
			if p.Port == 0 { p.Port = 3306 }
			if p.Name == "" { p.Name = "MySQL Database" }
		},
		"jest-watch": func(p *CreateProcessParams) {
			if p.Command == "" { p.Command = "npm" }
			if len(p.Args) == 0 { p.Args = []string{"run", "test:watch"} }
			if p.Type == "" { p.Type = domain.ProcessTypeTest }
			if p.Name == "" { p.Name = "Jest Test Watcher" }
		},
	}
	
	template, exists := templates[params.Template]
	if !exists {
		availableTemplates := make([]string, 0, len(templates))
		for name := range templates {
			availableTemplates = append(availableTemplates, name)
		}
		return fmt.Errorf("unknown template '%s'. Available templates: %v", params.Template, availableTemplates)
	}
	
	template(params)
	return nil
}

type ProcessIDParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handleProcessStart(params json.RawMessage) (interface{}, error) {
	var p ProcessIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if err := s.processService.Start(p.ID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "started"}, nil
}

func (s *MCPServer) handleProcessStop(params json.RawMessage) (interface{}, error) {
	var p ProcessIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if err := s.processService.Stop(p.ID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "stopped"}, nil
}

type ListProcessesParams struct {
	ProjectID *string             `json:"projectId,omitempty"`
	Status    *domain.ProcessStatus `json:"status,omitempty"`
	Type      *domain.ProcessType   `json:"type,omitempty"`
}

func (s *MCPServer) handleProcessList(params json.RawMessage) (interface{}, error) {
	var p ListProcessesParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == nil {
		current, err := s.projectService.GetCurrent()
		if err == nil {
			projectID = &current.ID
		}
	}
	
	filter := domain.ProcessFilter{
		ProjectID: projectID,
		Status:    p.Status,
		Type:      p.Type,
	}
	
	processes, err := s.processService.List(filter)
	if err != nil {
		return nil, err
	}
	
	// Return formatted markdown
	return FormatProcessesAsMarkdown(processes), nil
}

func (s *MCPServer) handleProcessGet(params json.RawMessage) (interface{}, error) {
	var p ProcessIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.processService.Get(p.ID)
}

type ProcessLogsParams struct {
	ID    string `json:"id"`
	Limit int    `json:"limit,omitempty"`
}

func (s *MCPServer) handleProcessLogs(params json.RawMessage) (interface{}, error) {
	var p ProcessLogsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if p.Limit == 0 {
		p.Limit = 100
	}
	
	logs, err := s.processService.GetLogs(p.ID, p.Limit)
	if err != nil {
		return nil, err
	}
	
	// Return formatted markdown
	return FormatProcessLogsAsMarkdown(logs), nil
}

func (s *MCPServer) handleProcessStatus(params json.RawMessage) (interface{}, error) {
	var p ProcessIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	process, err := s.processService.Get(p.ID)
	if err != nil {
		return nil, err
	}
	
	// Return formatted process status
	return FormatProcessStatusAsMarkdown(process), nil
}

type UpdateProcessParams struct {
	ID      string                 `json:"id"`
	Updates map[string]interface{} `json:"updates"`
}

func (s *MCPServer) handleProcessUpdate(params json.RawMessage) (interface{}, error) {
	var p UpdateProcessParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return s.processService.Update(p.ID, p.Updates)
}

type CreateProcessGroupParams struct {
	ProjectID   string   `json:"projectId,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ProcessIDs  []string `json:"processIds,omitempty"`
}

func (s *MCPServer) handleProcessGroupCreate(params json.RawMessage) (interface{}, error) {
	var p CreateProcessGroupParams
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
	
	group := domain.NewProcessGroup(projectID, p.Name, p.Description)
	if len(p.ProcessIDs) > 0 {
		group.ProcessIDs = p.ProcessIDs
	}
	
	if err := s.processService.CreateGroup(group); err != nil {
		return nil, err
	}
	
	return group, nil
}

type ProcessGroupIDParams struct {
	ID string `json:"id"`
}

func (s *MCPServer) handleProcessGroupStart(params json.RawMessage) (interface{}, error) {
	var p ProcessGroupIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if err := s.processService.StartGroup(p.ID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "group started"}, nil
}

func (s *MCPServer) handleProcessGroupStop(params json.RawMessage) (interface{}, error) {
	var p ProcessGroupIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	if err := s.processService.StopGroup(p.ID); err != nil {
		return nil, err
	}
	
	return map[string]string{"status": "group stopped"}, nil
}

// TODO handlers
type CreateTodoParams struct {
	ProjectID string               `json:"projectId,omitempty"`
	Card      *CreateTodoCard      `json:"card"`
	Context   *CreateTodoContext   `json:"context"`
	Criteria  *CreateTodoCriteria  `json:"criteria"`
}

type CreateTodoCard struct {
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	Priority       domain.Priority   `json:"priority,omitempty"`
	DueDate        *time.Time        `json:"dueDate,omitempty"`
	EstimatedHours *float64          `json:"estimatedHours,omitempty"`
	Labels         []string          `json:"labels,omitempty"`
	AssignedTo     *string           `json:"assignedTo,omitempty"`
}

type CreateTodoContext struct {
	Files        []string `json:"files,omitempty"`
	Dependencies []string `json:"dependencies"`
	Assumptions  []string `json:"assumptions"`
}

type CreateTodoCriteria struct {
	Acceptance   []string `json:"acceptance"`
	Verification []string `json:"verification,omitempty"`
}

type QuickTodoParams struct {
	ProjectID   string      `json:"projectId,omitempty"`
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
	Priority    string      `json:"priority,omitempty"`
	DueDate     *time.Time  `json:"dueDate,omitempty"`
	Labels      []string    `json:"labels,omitempty"`
	AssignedTo  string      `json:"assignedTo,omitempty"`
}

func (s *MCPServer) handleTodoQuickCreate(params json.RawMessage) (interface{}, error) {
	var p QuickTodoParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Validate required parameters
	if p.Title == "" {
		return nil, fmt.Errorf("title is required")
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
	
	// Set default priority if not specified
	var priority domain.Priority
	if p.Priority == "" {
		priority = domain.PriorityMedium
	} else {
		priority = domain.Priority(p.Priority)
	}
	
	// Set default description if not provided
	description := p.Description
	if description == "" {
		description = "Quick todo item - details to be added later"
	}
	
	// Helper for optional string pointer
	var assignedTo *string
	if p.AssignedTo != "" {
		assignedTo = &p.AssignedTo
	}
	
	// Create minimal 3 C's structure
	card := &CreateTodoCard{
		Title:          p.Title,
		Description:    description,
		Priority:       priority,
		DueDate:        p.DueDate,
		Labels:         p.Labels,
		AssignedTo:     assignedTo,
		EstimatedHours: nil, // Default to nil for quick todos
	}
	
	context := &CreateTodoContext{
		Files:        []string{}, // Empty files for quick todos
		Dependencies: []string{}, // Empty dependencies for quick todos
		Assumptions:  []string{"This is a quick todo - assumptions to be refined as needed"}, // Default assumption
	}
	
	criteria := &CreateTodoCriteria{
		Acceptance:   []string{"Task completed as described in title", "Implementation meets basic requirements"}, // Default acceptance criteria
		Verification: []string{"Manual verification of completion"}, // Default verification
	}
	
	// Create the task using the existing create logic
	fullParams := CreateTodoParams{
		ProjectID: projectID,
		Card:      card,
		Context:   context,
		Criteria:  criteria,
	}
	
	// Convert back to JSON and call the full create handler
	fullParamsJSON, err := json.Marshal(fullParams)
	if err != nil {
		return nil, fmt.Errorf("failed to convert quick todo to full structure: %w", err)
	}
	
	return s.handleTodoCreate(fullParamsJSON)
}

func (s *MCPServer) handleTodoCreate(params json.RawMessage) (interface{}, error) {
	var p CreateTodoParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Validate required structures
	if p.Card == nil {
		return nil, fmt.Errorf("card is required")
	}
	if p.Context == nil {
		return nil, fmt.Errorf("context is required")
	}
	if p.Criteria == nil {
		return nil, fmt.Errorf("criteria is required")
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
	
	// Set default priority if not specified
	priority := p.Card.Priority
	if priority == "" {
		priority = domain.PriorityMedium
	}
	
	// Create the TODO with the Card information
	todo := domain.NewTODO(projectID, p.Card.Title, p.Card.Description, priority)
	
	// Apply Card fields
	if p.Card.DueDate != nil {
		todo.SetDueDate(*p.Card.DueDate)
	}
	if p.Card.EstimatedHours != nil {
		todo.Card.EstimatedHours = p.Card.EstimatedHours
	}
	if len(p.Card.Labels) > 0 {
		todo.Card.Labels = p.Card.Labels
	}
	if p.Card.AssignedTo != nil {
		todo.Card.AssignedTo = p.Card.AssignedTo
	}
	
	// Apply Context fields
	if len(p.Context.Files) > 0 {
		todo.Context.Files = p.Context.Files
	}
	todo.Context.Dependencies = p.Context.Dependencies
	todo.Context.Assumptions = p.Context.Assumptions
	
	// Apply Criteria fields
	todo.Criteria.Acceptance = p.Criteria.Acceptance
	if len(p.Criteria.Verification) > 0 {
		todo.Criteria.Verification = p.Criteria.Verification
	}
	
	if err := s.taskService.Create(todo); err != nil {
		return nil, err
	}
	
	return todo, nil
}

type TodoIDParams struct {
	ID string `json:"id"`
}

type VerificationEvidenceInput struct {
	Evidence        string `json:"evidence"`
	TestType        string `json:"testType,omitempty"`
	TestResults     string `json:"testResults,omitempty"`
	RelatedCriteria []int  `json:"relatedCriteria,omitempty"`
}

type CompleteTodoParams struct {
	ID              string                      `json:"id"`
	Evidence        []VerificationEvidenceInput `json:"evidence"`
	CompletionNotes string                     `json:"completionNotes,omitempty"`
}

func (s *MCPServer) handleTodoComplete(params json.RawMessage) (interface{}, error) {
	var p CompleteTodoParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	// Get the task
	todo, err := s.taskService.Get(p.ID)
	if err != nil {
		return nil, err
	}
	
	// Convert input evidence to domain evidence with audit trail
	evidence := make([]domain.VerificationEvidence, len(p.Evidence))
	for i, e := range p.Evidence {
		evidence[i] = domain.VerificationEvidence{
			Evidence:        e.Evidence,
			TestedAt:        time.Now(),
			TestType:        e.TestType,
			TestResults:     e.TestResults,
			RelatedCriteria: e.RelatedCriteria,
			CommitHash:      s.getCurrentCommitHash(),
			FilesAffected:   s.getCurrentWorkingFiles(),
		}
	}
	
	// Complete with verification
	if err := todo.CompleteWithVerification(evidence, "compass-agent", p.CompletionNotes); err != nil {
		return nil, fmt.Errorf("failed to complete task with verification: %w", err)
	}
	
	// Update the task in storage
	updates := map[string]interface{}{
		"status":       todo.Card.Status,
		"completedAt":  todo.Card.CompletedAt,
		"updatedAt":    todo.Card.UpdatedAt,
		"verification": todo.Card.Verification,
	}
	
	return s.taskService.Update(p.ID, updates)
}

// Helper methods for audit trail capture
func (s *MCPServer) getCurrentCommitHash() string {
	// Try to get current git commit hash
	if output, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

func (s *MCPServer) getCurrentWorkingFiles() []string {
	// Get list of modified/staged files in current directory
	var files []string
	
	// Get modified files
	if output, err := exec.Command("git", "diff", "--name-only").Output(); err == nil {
		for _, file := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if file != "" {
				files = append(files, file)
			}
		}
	}
	
	// Get staged files
	if output, err := exec.Command("git", "diff", "--cached", "--name-only").Output(); err == nil {
		for _, file := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if file != "" {
				// Avoid duplicates
				found := false
				for _, existing := range files {
					if existing == file {
						found = true
						break
					}
				}
				if !found {
					files = append(files, file)
				}
			}
		}
	}
	
	return files
}

func (s *MCPServer) handleTodoReopen(params json.RawMessage) (interface{}, error) {
	var p TodoIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	todo, err := s.taskService.Get(p.ID)
	if err != nil {
		return nil, err
	}
	
	todo.Reopen()
	
	updates := map[string]interface{}{
		"status":      todo.Card.Status,
		"completedAt": nil,
		"updatedAt":   todo.Card.UpdatedAt,
	}
	
	return s.taskService.Update(p.ID, updates)
}

type ListTodosParams struct {
	ProjectID    *string           `json:"projectId,omitempty"`
	Status       *domain.TaskStatus `json:"status,omitempty"`
	Priority     *domain.Priority   `json:"priority,omitempty"`
	Labels       []string          `json:"labels,omitempty"`
	AssignedTo   *string           `json:"assignedTo,omitempty"`
	DueBefore    *time.Time        `json:"dueBefore,omitempty"`
	DueAfter     *time.Time        `json:"dueAfter,omitempty"`
	Limit        int               `json:"limit,omitempty"`
}

func (s *MCPServer) handleTodoList(params json.RawMessage) (interface{}, error) {
	var p ListTodosParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == nil {
		current, err := s.projectService.GetCurrent()
		if err == nil {
			projectID = &current.ID
		}
	}
	
	filter := domain.TaskFilter{
		ProjectID:  projectID,
		Status:     p.Status,
		Priority:   p.Priority,
		Labels:     p.Labels,
		AssignedTo: p.AssignedTo,
		DueBefore:  p.DueBefore,
		DueAfter:   p.DueAfter,
	}
	
	todos, err := s.taskService.List(filter)
	if err != nil {
		return nil, err
	}
	
	// Apply limit if specified
	if p.Limit > 0 && len(todos) > p.Limit {
		todos = todos[:p.Limit]
	}
	
	// Return markdown formatted string
	return FormatTodosAsMarkdown(todos), nil
}

type OverdueTodosParams struct {
	ProjectID *string `json:"projectId,omitempty"`
}

func (s *MCPServer) handleTodoOverdue(params json.RawMessage) (interface{}, error) {
	var p OverdueTodosParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, fmt.Errorf("invalid parameters: %w", err)
		}
	}
	
	// Use current project if not specified
	projectID := p.ProjectID
	if projectID == nil {
		current, err := s.projectService.GetCurrent()
		if err == nil {
			projectID = &current.ID
		}
	}
	
	// For now, get all tasks and filter manually (can be optimized later)
	allTasks, err := s.taskService.List(domain.TaskFilter{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	
	var overdueTasks []*domain.Task
	for _, task := range allTasks {
		if task.IsOverdue() {
			overdueTasks = append(overdueTasks, task)
		}
	}
	
	return overdueTasks, nil
}

type UpdateTodoPriorityParams struct {
	ID       string          `json:"id"`
	Priority domain.Priority `json:"priority"`
}

func (s *MCPServer) handleTodoUpdatePriority(params json.RawMessage) (interface{}, error) {
	var p UpdateTodoPriorityParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	updates := map[string]interface{}{
		"priority":  p.Priority,
		"updatedAt": time.Now(),
	}
	
	return s.taskService.Update(p.ID, updates)
}

type SetTodoDueParams struct {
	ID      string     `json:"id"`
	DueDate *time.Time `json:"dueDate"`
}

func (s *MCPServer) handleTodoSetDue(params json.RawMessage) (interface{}, error) {
	var p SetTodoDueParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	updates := map[string]interface{}{
		"dueDate":   p.DueDate,
		"updatedAt": time.Now(),
	}
	
	return s.taskService.Update(p.ID, updates)
}

type TodoLabelParams struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (s *MCPServer) handleTodoAddLabel(params json.RawMessage) (interface{}, error) {
	var p TodoLabelParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	todo, err := s.taskService.Get(p.ID)
	if err != nil {
		return nil, err
	}
	
	todo.AddLabel(p.Label)
	
	updates := map[string]interface{}{
		"labels":    todo.Card.Labels,
		"updatedAt": todo.Card.UpdatedAt,
	}
	
	return s.taskService.Update(p.ID, updates)
}

func (s *MCPServer) handleTodoRemoveLabel(params json.RawMessage) (interface{}, error) {
	var p TodoLabelParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	todo, err := s.taskService.Get(p.ID)
	if err != nil {
		return nil, err
	}
	
	todo.RemoveLabel(p.Label)
	
	updates := map[string]interface{}{
		"labels":    todo.Card.Labels,
		"updatedAt": todo.Card.UpdatedAt,
	}
	
	return s.taskService.Update(p.ID, updates)
}

type UpdateTodoProgressParams struct {
	ID    string  `json:"id"`
	Hours float64 `json:"hours"`
}

func (s *MCPServer) handleTodoUpdateProgress(params json.RawMessage) (interface{}, error) {
	var p UpdateTodoProgressParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	todo, err := s.taskService.Get(p.ID)
	if err != nil {
		return nil, err
	}
	
	todo.UpdateProgress(p.Hours)
	
	updates := map[string]interface{}{
		"actualHours": todo.Card.ActualHours,
		"updatedAt":   todo.Card.UpdatedAt,
	}
	
	return s.taskService.Update(p.ID, updates)
}

// Shutdown gracefully shuts down the MCP server and all managed processes
func (s *MCPServer) Shutdown() {
	log.Printf("MCPServer: Shutdown called")
	if s.processService != nil {
		log.Printf("MCPServer: Calling processService.Shutdown()")
		s.processService.Shutdown()
		log.Printf("MCPServer: processService.Shutdown() completed")
	} else {
		log.Printf("MCPServer: processService is nil!")
	}
}