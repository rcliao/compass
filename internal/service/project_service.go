package service

import (
	"github.com/rcliao/compass/internal/domain"
)

type ProjectService struct {
	storage ProjectStorage
}

type ProjectStorage interface {
	CreateProject(project *domain.Project) error
	GetProject(id string) (*domain.Project, error)
	ListProjects() ([]*domain.Project, error)
	SetCurrentProject(id string) error
	GetCurrentProject() (*domain.Project, error)
}

func NewProjectService(storage ProjectStorage) *ProjectService {
	return &ProjectService{
		storage: storage,
	}
}

func (s *ProjectService) Create(project *domain.Project) error {
	return s.storage.CreateProject(project)
}

func (s *ProjectService) Get(id string) (*domain.Project, error) {
	return s.storage.GetProject(id)
}

func (s *ProjectService) List() ([]*domain.Project, error) {
	return s.storage.ListProjects()
}

func (s *ProjectService) SetCurrent(id string) error {
	return s.storage.SetCurrentProject(id)
}

func (s *ProjectService) GetCurrent() (*domain.Project, error) {
	return s.storage.GetCurrentProject()
}