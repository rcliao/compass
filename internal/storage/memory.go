package storage

import (
	"fmt"
	"sync"

	"github.com/rcliao/compass/internal/domain"
)

type MemoryStorage struct {
	mu           sync.RWMutex
	tasks        map[string]*domain.Task
	projects     map[string]*domain.Project
	discoveries  map[string]*domain.Discovery
	decisions    map[string]*domain.Decision
	currentProject *string
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tasks:       make(map[string]*domain.Task),
		projects:    make(map[string]*domain.Project),
		discoveries: make(map[string]*domain.Discovery),
		decisions:   make(map[string]*domain.Decision),
	}
}

// Task Repository Implementation
func (ms *MemoryStorage) CreateTask(task *domain.Task) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if _, exists := ms.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}
	
	ms.tasks[task.ID] = task
	return nil
}

func (ms *MemoryStorage) UpdateTask(id string, updates map[string]interface{}) (*domain.Task, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	task, exists := ms.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", id)
	}
	
	// Create a copy to avoid modifying the original
	updatedTask := *task
	
	// Apply updates (simplified for now - in a real implementation, you'd handle all fields)
	if title, ok := updates["title"].(string); ok {
		updatedTask.Card.Title = title
	}
	if description, ok := updates["description"].(string); ok {
		updatedTask.Card.Description = description
	}
	if status, ok := updates["status"].(domain.TaskStatus); ok {
		updatedTask.Card.Status = status
	}
	
	ms.tasks[id] = &updatedTask
	return &updatedTask, nil
}

func (ms *MemoryStorage) GetTask(id string) (*domain.Task, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	task, exists := ms.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task with ID %s not found", id)
	}
	
	return task, nil
}

func (ms *MemoryStorage) ListTasks(filter domain.TaskFilter) ([]*domain.Task, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	var result []*domain.Task
	
	for _, task := range ms.tasks {
		if filter.ProjectID != nil && task.ProjectID != *filter.ProjectID {
			continue
		}
		if filter.Status != nil && task.Card.Status != *filter.Status {
			continue
		}
		if filter.Parent != nil && task.Card.Parent != filter.Parent {
			continue
		}
		
		result = append(result, task)
	}
	
	return result, nil
}

func (ms *MemoryStorage) DeleteTask(id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if _, exists := ms.tasks[id]; !exists {
		return fmt.Errorf("task with ID %s not found", id)
	}
	
	delete(ms.tasks, id)
	return nil
}

// Project Repository Implementation
func (ms *MemoryStorage) CreateProject(project *domain.Project) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if _, exists := ms.projects[project.ID]; exists {
		return fmt.Errorf("project with ID %s already exists", project.ID)
	}
	
	ms.projects[project.ID] = project
	return nil
}

func (ms *MemoryStorage) GetProject(id string) (*domain.Project, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	project, exists := ms.projects[id]
	if !exists {
		return nil, fmt.Errorf("project with ID %s not found", id)
	}
	
	return project, nil
}

func (ms *MemoryStorage) ListProjects() ([]*domain.Project, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	var result []*domain.Project
	for _, project := range ms.projects {
		result = append(result, project)
	}
	
	return result, nil
}

func (ms *MemoryStorage) SetCurrentProject(id string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if _, exists := ms.projects[id]; !exists {
		return fmt.Errorf("project with ID %s not found", id)
	}
	
	ms.currentProject = &id
	return nil
}

func (ms *MemoryStorage) GetCurrentProject() (*domain.Project, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	if ms.currentProject == nil {
		return nil, fmt.Errorf("no current project set")
	}
	
	project, exists := ms.projects[*ms.currentProject]
	if !exists {
		return nil, fmt.Errorf("current project with ID %s not found", *ms.currentProject)
	}
	
	return project, nil
}

// Discovery and Decision storage (simplified for now)
func (ms *MemoryStorage) CreateDiscovery(discovery *domain.Discovery) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	ms.discoveries[discovery.ID] = discovery
	return nil
}

func (ms *MemoryStorage) CreateDecision(decision *domain.Decision) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	ms.decisions[decision.ID] = decision
	return nil
}