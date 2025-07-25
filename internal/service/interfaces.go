package service

import (
	"github.com/rcliao/compass/internal/domain"
)

// ProcessStorage interface for process data persistence
type ProcessStorage interface {
	SaveProcess(projectID string, process *domain.Process) error
	GetProcess(processID string) (*domain.Process, error)
	ListProcesses(filter domain.ProcessFilter) ([]*domain.Process, error)
	SaveProcessGroup(projectID string, group *domain.ProcessGroup) error
	GetProcessGroup(groupID string) (*domain.ProcessGroup, error)
	SaveProcessLogs(logs []*domain.ProcessLog) error
	GetProcessLogs(processID string, limit int) ([]*domain.ProcessLog, error)
}