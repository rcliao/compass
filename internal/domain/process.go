package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProcessStatus represents the current state of a process
type ProcessStatus string

const (
	ProcessStatusPending  ProcessStatus = "pending"
	ProcessStatusStarting ProcessStatus = "starting"
	ProcessStatusRunning  ProcessStatus = "running"
	ProcessStatusStopping ProcessStatus = "stopping"
	ProcessStatusStopped  ProcessStatus = "stopped"
	ProcessStatusFailed   ProcessStatus = "failed"
	ProcessStatusCrashed  ProcessStatus = "crashed"
)

// ProcessType categorizes different kinds of processes
type ProcessType string

const (
	ProcessTypeWebServer   ProcessType = "web-server"
	ProcessTypeAPIServer   ProcessType = "api-server"
	ProcessTypeBuildTool   ProcessType = "build-tool"
	ProcessTypeWatcher     ProcessType = "watcher"
	ProcessTypeTest        ProcessType = "test"
	ProcessTypeDatabase    ProcessType = "database"
	ProcessTypeCustom      ProcessType = "custom"
)

// Process represents a managed subprocess
type Process struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"projectId"`
	Name        string                 `json:"name"`
	Type        ProcessType            `json:"type"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args,omitempty"`
	WorkingDir  string                 `json:"workingDir,omitempty"`
	Environment map[string]string      `json:"environment,omitempty"`
	Status      ProcessStatus          `json:"status"`
	PID         int                    `json:"pid,omitempty"`
	Port        int                    `json:"port,omitempty"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	StoppedAt   *time.Time             `json:"stoppedAt,omitempty"`
	LastHealthCheck *time.Time         `json:"lastHealthCheck,omitempty"`
	HealthStatus string                `json:"healthStatus,omitempty"`
	RestartPolicy RestartPolicy        `json:"restartPolicy"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

// RestartPolicy defines how a process should be restarted
type RestartPolicy struct {
	Enabled     bool          `json:"enabled"`
	MaxRetries  int           `json:"maxRetries"`
	RetryDelay  time.Duration `json:"retryDelay"`
	RetryCount  int           `json:"retryCount"`
	LastRestart *time.Time    `json:"lastRestart,omitempty"`
}

// ProcessGroup represents a collection of related processes
type ProcessGroup struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ProcessIDs  []string  `json:"processIds"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ProcessLog represents captured output from a process
type ProcessLog struct {
	ID        string     `json:"id"`
	ProcessID string     `json:"processId"`
	Type      LogType    `json:"type"`
	Message   string     `json:"message"`
	Timestamp time.Time  `json:"timestamp"`
	Level     LogLevel   `json:"level,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogType distinguishes between stdout and stderr
type LogType string

const (
	LogTypeStdout LogType = "stdout"
	LogTypeStderr LogType = "stderr"
	LogTypeSystem LogType = "system"
)

// LogLevel represents log severity
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

// ProcessFilter for querying processes
type ProcessFilter struct {
	ProjectID *string        `json:"projectId,omitempty"`
	Status    *ProcessStatus `json:"status,omitempty"`
	Type      *ProcessType   `json:"type,omitempty"`
	GroupID   *string        `json:"groupId,omitempty"`
}

// ProcessHealthCheck defines health check configuration
type ProcessHealthCheck struct {
	Enabled       bool          `json:"enabled"`
	Type          HealthCheckType `json:"type"`
	Interval      time.Duration `json:"interval"`
	Timeout       time.Duration `json:"timeout"`
	Endpoint      string        `json:"endpoint,omitempty"`
	ExpectedCode  int           `json:"expectedCode,omitempty"`
	MaxFailures   int           `json:"maxFailures"`
	FailureCount  int           `json:"failureCount"`
}

// HealthCheckType defines different health check methods
type HealthCheckType string

const (
	HealthCheckTypeHTTP    HealthCheckType = "http"
	HealthCheckTypeTCP     HealthCheckType = "tcp"
	HealthCheckTypeProcess HealthCheckType = "process"
	HealthCheckTypeCustom  HealthCheckType = "custom"
)

// NewProcess creates a new Process instance
func NewProcess(projectID, name, command string, args []string) *Process {
	now := time.Now()
	return &Process{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Name:      name,
		Command:   command,
		Args:      args,
		Status:    ProcessStatusPending,
		Type:      ProcessTypeCustom,
		RestartPolicy: RestartPolicy{
			Enabled:    false,
			MaxRetries: 3,
			RetryDelay: 5 * time.Second,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewProcessGroup creates a new ProcessGroup instance
func NewProcessGroup(projectID, name, description string) *ProcessGroup {
	now := time.Now()
	return &ProcessGroup{
		ID:          uuid.New().String(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		ProcessIDs:  []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewProcessLog creates a new ProcessLog entry
func NewProcessLog(processID string, logType LogType, message string) *ProcessLog {
	return &ProcessLog{
		ID:        uuid.New().String(),
		ProcessID: processID,
		Type:      logType,
		Message:   message,
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
	}
}

// IsRunning checks if the process is in a running state
func (p *Process) IsRunning() bool {
	return p.Status == ProcessStatusRunning || p.Status == ProcessStatusStarting
}

// CanStart checks if the process can be started
func (p *Process) CanStart() bool {
	return p.Status == ProcessStatusPending || p.Status == ProcessStatusStopped || 
	       p.Status == ProcessStatusFailed || p.Status == ProcessStatusCrashed
}

// CanStop checks if the process can be stopped
func (p *Process) CanStop() bool {
	return p.Status == ProcessStatusRunning || p.Status == ProcessStatusStarting
}

// Duration returns how long the process has been running
func (p *Process) Duration() time.Duration {
	if p.StartedAt == nil {
		return 0
	}
	if p.StoppedAt != nil {
		return p.StoppedAt.Sub(*p.StartedAt)
	}
	if p.IsRunning() {
		return time.Since(*p.StartedAt)
	}
	return 0
}