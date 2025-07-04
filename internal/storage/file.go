package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rcliao/compass/internal/domain"
)

type FileStorage struct {
	basePath string
	mu       sync.RWMutex
}

type Config struct {
	CurrentProject *string `json:"currentProject,omitempty"`
}

func NewFileStorage(basePath string) (*FileStorage, error) {
	fs := &FileStorage{
		basePath: basePath,
	}
	
	err := fs.initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize file storage: %w", err)
	}
	
	return fs, nil
}

func (fs *FileStorage) initialize() error {
	// Create .compass directory
	compassDir := filepath.Join(fs.basePath, ".compass")
	if err := os.MkdirAll(compassDir, 0755); err != nil {
		return err
	}
	
	// Create projects directory
	projectsDir := filepath.Join(compassDir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		return err
	}
	
	// Create config.json if it doesn't exist
	configPath := filepath.Join(compassDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := Config{}
		return fs.saveJSON(configPath, config)
	}
	
	return nil
}

func (fs *FileStorage) projectDir(projectID string) string {
	return filepath.Join(fs.basePath, ".compass", "projects", projectID)
}

func (fs *FileStorage) ensureProjectDir(projectID string) error {
	dir := fs.projectDir(projectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Create subdirectories
	subdirs := []string{"planning", "index"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(dir, subdir), 0755); err != nil {
			return err
		}
	}
	
	return nil
}

func (fs *FileStorage) saveJSON(path string, data interface{}) error {
	tempPath := path + ".tmp"
	
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		os.Remove(tempPath)
		return err
	}
	
	return os.Rename(tempPath, path)
}

func (fs *FileStorage) loadJSON(path string, target interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return json.NewDecoder(file).Decode(target)
}

// Task Repository Implementation
func (fs *FileStorage) CreateTask(task *domain.Task) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := fs.ensureProjectDir(task.ProjectID); err != nil {
		return err
	}
	
	tasks, err := fs.loadTasks(task.ProjectID)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	
	// Check if task already exists
	for _, t := range tasks {
		if t.ID == task.ID {
			return fmt.Errorf("task with ID %s already exists", task.ID)
		}
	}
	
	tasks = append(tasks, task)
	return fs.saveTasks(task.ProjectID, tasks)
}

func (fs *FileStorage) loadTasks(projectID string) ([]*domain.Task, error) {
	tasksPath := filepath.Join(fs.projectDir(projectID), "tasks.json")
	
	var tasks []*domain.Task
	err := fs.loadJSON(tasksPath, &tasks)
	if os.IsNotExist(err) {
		return make([]*domain.Task, 0), nil
	}
	
	return tasks, err
}

func (fs *FileStorage) saveTasks(projectID string, tasks []*domain.Task) error {
	tasksPath := filepath.Join(fs.projectDir(projectID), "tasks.json")
	return fs.saveJSON(tasksPath, tasks)
}

func (fs *FileStorage) UpdateTask(id string, updates map[string]interface{}) (*domain.Task, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	// Find the task across all projects (for simplicity)
	projects, err := fs.ListProjects()
	if err != nil {
		return nil, err
	}
	
	for _, project := range projects {
		tasks, err := fs.loadTasks(project.ID)
		if err != nil {
			continue
		}
		
		for i, task := range tasks {
			if task.ID == id {
				// Create a copy and apply updates
				updatedTask := *task
				
				if title, ok := updates["title"].(string); ok {
					updatedTask.Card.Title = title
				}
				if description, ok := updates["description"].(string); ok {
					updatedTask.Card.Description = description
				}
				if status, ok := updates["status"].(domain.TaskStatus); ok {
					updatedTask.Card.Status = status
				}
				
				tasks[i] = &updatedTask
				err = fs.saveTasks(project.ID, tasks)
				if err != nil {
					return nil, err
				}
				
				return &updatedTask, nil
			}
		}
	}
	
	return nil, fmt.Errorf("task with ID %s not found", id)
}

func (fs *FileStorage) GetTask(id string) (*domain.Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	projects, err := fs.ListProjects()
	if err != nil {
		return nil, err
	}
	
	for _, project := range projects {
		tasks, err := fs.loadTasks(project.ID)
		if err != nil {
			continue
		}
		
		for _, task := range tasks {
			if task.ID == id {
				return task, nil
			}
		}
	}
	
	return nil, fmt.Errorf("task with ID %s not found", id)
}

func (fs *FileStorage) ListTasks(filter domain.TaskFilter) ([]*domain.Task, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	var result []*domain.Task
	
	if filter.ProjectID != nil {
		tasks, err := fs.loadTasks(*filter.ProjectID)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		result = tasks
	} else {
		// Load tasks from all projects
		projects, err := fs.ListProjects()
		if err != nil {
			return nil, err
		}
		
		for _, project := range projects {
			tasks, err := fs.loadTasks(project.ID)
			if err != nil && !os.IsNotExist(err) {
				continue
			}
			result = append(result, tasks...)
		}
	}
	
	// Apply filters
	var filtered []*domain.Task
	for _, task := range result {
		if filter.Status != nil && task.Card.Status != *filter.Status {
			continue
		}
		if filter.Parent != nil && task.Card.Parent != filter.Parent {
			continue
		}
		
		filtered = append(filtered, task)
	}
	
	return filtered, nil
}

func (fs *FileStorage) DeleteTask(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	projects, err := fs.ListProjects()
	if err != nil {
		return err
	}
	
	for _, project := range projects {
		tasks, err := fs.loadTasks(project.ID)
		if err != nil {
			continue
		}
		
		for i, task := range tasks {
			if task.ID == id {
				// Remove task from slice
				tasks = append(tasks[:i], tasks[i+1:]...)
				return fs.saveTasks(project.ID, tasks)
			}
		}
	}
	
	return fmt.Errorf("task with ID %s not found", id)
}

// Project Repository Implementation
func (fs *FileStorage) CreateProject(project *domain.Project) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := fs.ensureProjectDir(project.ID); err != nil {
		return err
	}
	
	projectPath := filepath.Join(fs.projectDir(project.ID), "project.json")
	if _, err := os.Stat(projectPath); err == nil {
		return fmt.Errorf("project with ID %s already exists", project.ID)
	}
	
	return fs.saveJSON(projectPath, project)
}

func (fs *FileStorage) GetProject(id string) (*domain.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	projectPath := filepath.Join(fs.projectDir(id), "project.json")
	
	var project domain.Project
	err := fs.loadJSON(projectPath, &project)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("project with ID %s not found", id)
	}
	
	return &project, err
}

func (fs *FileStorage) ListProjects() ([]*domain.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	projectsDir := filepath.Join(fs.basePath, ".compass", "projects")
	
	var projects []*domain.Project
	
	err := filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() && info.Name() != "projects" {
			projectPath := filepath.Join(path, "project.json")
			if _, err := os.Stat(projectPath); err == nil {
				var project domain.Project
				if err := fs.loadJSON(projectPath, &project); err == nil {
					projects = append(projects, &project)
				}
			}
		}
		
		return nil
	})
	
	return projects, err
}

func (fs *FileStorage) SetCurrentProject(id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	// Verify project exists
	if _, err := fs.GetProject(id); err != nil {
		return err
	}
	
	configPath := filepath.Join(fs.basePath, ".compass", "config.json")
	config := Config{CurrentProject: &id}
	
	return fs.saveJSON(configPath, config)
}

func (fs *FileStorage) GetCurrentProject() (*domain.Project, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	configPath := filepath.Join(fs.basePath, ".compass", "config.json")
	
	var config Config
	if err := fs.loadJSON(configPath, &config); err != nil {
		return nil, err
	}
	
	if config.CurrentProject == nil {
		return nil, fmt.Errorf("no current project set")
	}
	
	return fs.GetProject(*config.CurrentProject)
}

// Planning Storage Implementation
func (fs *FileStorage) CreatePlanningSession(session *domain.PlanningSession) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := fs.ensureProjectDir(session.ProjectID); err != nil {
		return err
	}
	
	// Load existing sessions
	sessions, err := fs.loadPlanningSessions(session.ProjectID)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	
	// Check if session already exists
	for _, s := range sessions {
		if s.ID == session.ID {
			return fmt.Errorf("planning session with ID %s already exists", session.ID)
		}
	}
	
	sessions = append(sessions, session)
	return fs.savePlanningSessions(session.ProjectID, sessions)
}

func (fs *FileStorage) loadPlanningSessions(projectID string) ([]*domain.PlanningSession, error) {
	planningDir := filepath.Join(fs.projectDir(projectID), "planning")
	sessionsPath := filepath.Join(planningDir, "sessions.json")
	
	var sessions []*domain.PlanningSession
	err := fs.loadJSON(sessionsPath, &sessions)
	if os.IsNotExist(err) {
		return make([]*domain.PlanningSession, 0), nil
	}
	
	return sessions, err
}

func (fs *FileStorage) savePlanningSessions(projectID string, sessions []*domain.PlanningSession) error {
	planningDir := filepath.Join(fs.projectDir(projectID), "planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		return err
	}
	
	sessionsPath := filepath.Join(planningDir, "sessions.json")
	return fs.saveJSON(sessionsPath, sessions)
}

func (fs *FileStorage) GetPlanningSession(id string) (*domain.PlanningSession, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	projects, err := fs.ListProjects()
	if err != nil {
		return nil, err
	}
	
	for _, project := range projects {
		sessions, err := fs.loadPlanningSessions(project.ID)
		if err != nil {
			continue
		}
		
		for _, session := range sessions {
			if session.ID == id {
				return session, nil
			}
		}
	}
	
	return nil, fmt.Errorf("planning session with ID %s not found", id)
}

func (fs *FileStorage) ListPlanningSessions(projectID string) ([]*domain.PlanningSession, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	return fs.loadPlanningSessions(projectID)
}

func (fs *FileStorage) UpdatePlanningSession(id string, updates map[string]interface{}) (*domain.PlanningSession, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	projects, err := fs.ListProjects()
	if err != nil {
		return nil, err
	}
	
	for _, project := range projects {
		sessions, err := fs.loadPlanningSessions(project.ID)
		if err != nil {
			continue
		}
		
		for i, session := range sessions {
			if session.ID == id {
				// Create a copy and apply updates
				updatedSession := *session
				
				if status, ok := updates["status"].(domain.PlanningSessionStatus); ok {
					updatedSession.Status = status
				}
				if tasks, ok := updates["tasks"].([]string); ok {
					updatedSession.Tasks = tasks
				}
				
				sessions[i] = &updatedSession
				err = fs.savePlanningSessions(project.ID, sessions)
				if err != nil {
					return nil, err
				}
				
				return &updatedSession, nil
			}
		}
	}
	
	return nil, fmt.Errorf("planning session with ID %s not found", id)
}

// Discovery Storage Implementation
func (fs *FileStorage) CreateDiscovery(discovery *domain.Discovery) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := fs.ensureProjectDir(discovery.ProjectID); err != nil {
		return err
	}
	
	discoveries, err := fs.loadDiscoveries(discovery.ProjectID)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	
	discoveries = append(discoveries, discovery)
	return fs.saveDiscoveries(discovery.ProjectID, discoveries)
}

func (fs *FileStorage) loadDiscoveries(projectID string) ([]*domain.Discovery, error) {
	discoveriesPath := filepath.Join(fs.projectDir(projectID), "discoveries.json")
	
	var discoveries []*domain.Discovery
	err := fs.loadJSON(discoveriesPath, &discoveries)
	if os.IsNotExist(err) {
		return make([]*domain.Discovery, 0), nil
	}
	
	return discoveries, err
}

func (fs *FileStorage) saveDiscoveries(projectID string, discoveries []*domain.Discovery) error {
	discoveriesPath := filepath.Join(fs.projectDir(projectID), "discoveries.json")
	return fs.saveJSON(discoveriesPath, discoveries)
}

func (fs *FileStorage) ListDiscoveries(projectID string) ([]*domain.Discovery, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	return fs.loadDiscoveries(projectID)
}

// Decision Storage Implementation
func (fs *FileStorage) CreateDecision(decision *domain.Decision) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	if err := fs.ensureProjectDir(decision.ProjectID); err != nil {
		return err
	}
	
	decisions, err := fs.loadDecisions(decision.ProjectID)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	
	decisions = append(decisions, decision)
	return fs.saveDecisions(decision.ProjectID, decisions)
}

func (fs *FileStorage) loadDecisions(projectID string) ([]*domain.Decision, error) {
	decisionsPath := filepath.Join(fs.projectDir(projectID), "decisions.json")
	
	var decisions []*domain.Decision
	err := fs.loadJSON(decisionsPath, &decisions)
	if os.IsNotExist(err) {
		return make([]*domain.Decision, 0), nil
	}
	
	return decisions, err
}

func (fs *FileStorage) saveDecisions(projectID string, decisions []*domain.Decision) error {
	decisionsPath := filepath.Join(fs.projectDir(projectID), "decisions.json")
	return fs.saveJSON(decisionsPath, decisions)
}

func (fs *FileStorage) ListDecisions(projectID string) ([]*domain.Decision, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	
	return fs.loadDecisions(projectID)
}