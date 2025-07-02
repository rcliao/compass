package service

import (
	"github.com/rcliao/compass/internal/domain"
)

type TaskService struct {
	storage TaskStorage
}

type TaskStorage interface {
	CreateTask(task *domain.Task) error
	UpdateTask(id string, updates map[string]interface{}) (*domain.Task, error)
	GetTask(id string) (*domain.Task, error)
	ListTasks(filter domain.TaskFilter) ([]*domain.Task, error)
	DeleteTask(id string) error
}

func NewTaskService(storage TaskStorage) *TaskService {
	return &TaskService{
		storage: storage,
	}
}

func (s *TaskService) Create(task *domain.Task) error {
	return s.storage.CreateTask(task)
}

func (s *TaskService) Update(id string, updates map[string]interface{}) (*domain.Task, error) {
	return s.storage.UpdateTask(id, updates)
}

func (s *TaskService) Get(id string) (*domain.Task, error) {
	return s.storage.GetTask(id)
}

func (s *TaskService) List(filter domain.TaskFilter) ([]*domain.Task, error) {
	return s.storage.ListTasks(filter)
}

func (s *TaskService) Delete(id string) error {
	return s.storage.DeleteTask(id)
}