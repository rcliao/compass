package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

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

// ProcessService manages process lifecycle and operations
type ProcessService struct {
	storage           ProcessStorage
	processes         map[string]*ProcessInfo
	mu                sync.RWMutex
	logBuffers        map[string]*LogBuffer
	ctx               context.Context
	cancel            context.CancelFunc
	defaultWorkingDir string // Directory where MCP server was started (user's project dir)
}

// ProcessInfo wraps a process with its command and channels
type ProcessInfo struct {
	Process *domain.Process
	Cmd     *exec.Cmd
	Done    chan error
	Cancel  context.CancelFunc
}

// LogBuffer manages process logs with rotation
type LogBuffer struct {
	mu       sync.Mutex
	logs     []*domain.ProcessLog
	maxSize  int
	storage  ProcessStorage
}

// NewProcessService creates a new process service instance
func NewProcessService(storage ProcessStorage, defaultWorkingDir string) *ProcessService {
	ctx, cancel := context.WithCancel(context.Background())
	ps := &ProcessService{
		storage:           storage,
		processes:         make(map[string]*ProcessInfo),
		logBuffers:        make(map[string]*LogBuffer),
		ctx:               ctx,
		cancel:            cancel,
		defaultWorkingDir: defaultWorkingDir,
	}
	
	// Start health check routine
	go ps.healthCheckLoop()
	
	// Load existing processes from storage
	ps.loadProcesses()
	
	return ps
}

// Create creates a new process definition
func (ps *ProcessService) Create(process *domain.Process) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	// Set defaults - use the directory where MCP server was started (user's project dir)
	if process.WorkingDir == "" {
		process.WorkingDir = ps.defaultWorkingDir
	}
	
	// Save to storage
	if err := ps.storage.SaveProcess(process.ProjectID, process); err != nil {
		return fmt.Errorf("failed to save process: %w", err)
	}
	
	// Initialize log buffer
	ps.logBuffers[process.ID] = &LogBuffer{
		logs:    make([]*domain.ProcessLog, 0, 1000),
		maxSize: 10000,
		storage: ps.storage,
	}
	
	return nil
}

// Start starts a process
func (ps *ProcessService) Start(processID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	// Get process from storage
	process, err := ps.storage.GetProcess(processID)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}
	
	if !process.CanStart() {
		return fmt.Errorf("process cannot be started in status: %s", process.Status)
	}
	
	// Create command with context
	ctx, cancel := context.WithCancel(ps.ctx)
	cmd := exec.CommandContext(ctx, process.Command, process.Args...)
	
	// Set working directory
	cmd.Dir = process.WorkingDir
	
	// Set environment variables
	if len(process.Environment) > 0 {
		env := os.Environ()
		for k, v := range process.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}
	
	// Setup pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Update process status
	process.Status = domain.ProcessStatusStarting
	now := time.Now()
	process.StartedAt = &now
	process.UpdatedAt = now
	
	// Start the process
	if err := cmd.Start(); err != nil {
		process.Status = domain.ProcessStatusFailed
		ps.storage.SaveProcess(process.ProjectID, process)
		return fmt.Errorf("failed to start process: %w", err)
	}
	
	// Update with PID
	process.PID = cmd.Process.Pid
	process.Status = domain.ProcessStatusRunning
	ps.storage.SaveProcess(process.ProjectID, process)
	
	// Create process info
	done := make(chan error, 1)
	info := &ProcessInfo{
		Process: process,
		Cmd:     cmd,
		Done:    done,
		Cancel:  cancel,
	}
	ps.processes[processID] = info
	
	// Start log capture routines
	go ps.captureOutput(processID, stdout, domain.LogTypeStdout)
	go ps.captureOutput(processID, stderr, domain.LogTypeStderr)
	
	// Monitor process completion
	go func() {
		err := cmd.Wait()
		done <- err
		
		ps.mu.Lock()
		defer ps.mu.Unlock()
		
		// Update process status
		now := time.Now()
		process.StoppedAt = &now
		process.UpdatedAt = now
		
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == -1 {
					process.Status = domain.ProcessStatusCrashed
				} else {
					process.Status = domain.ProcessStatusFailed
				}
			} else {
				process.Status = domain.ProcessStatusFailed
			}
		} else {
			process.Status = domain.ProcessStatusStopped
		}
		
		ps.storage.SaveProcess(process.ProjectID, process)
		delete(ps.processes, processID)
		
		// Handle restart policy
		if process.RestartPolicy.Enabled && process.Status == domain.ProcessStatusCrashed {
			ps.handleRestart(process)
		}
	}()
	
	// Log start event
	ps.addLog(processID, domain.LogTypeSystem, fmt.Sprintf("Process started with PID %d", process.PID))
	
	return nil
}

// Stop stops a running process
func (ps *ProcessService) Stop(processID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	info, exists := ps.processes[processID]
	if !exists {
		return fmt.Errorf("process not running")
	}
	
	if !info.Process.CanStop() {
		return fmt.Errorf("process cannot be stopped in status: %s", info.Process.Status)
	}
	
	// Update status
	info.Process.Status = domain.ProcessStatusStopping
	ps.storage.SaveProcess(info.Process.ProjectID, info.Process)
	
	// Try graceful shutdown first
	if info.Cmd.Process != nil {
		// Send SIGTERM
		if err := info.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// If SIGTERM fails, try SIGKILL
			if err := info.Cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to stop process: %w", err)
			}
		}
	}
	
	// Cancel context
	info.Cancel()
	
	// Log stop event
	ps.addLog(processID, domain.LogTypeSystem, "Process stop requested")
	
	return nil
}

// GetLogs retrieves logs for a process
func (ps *ProcessService) GetLogs(processID string, limit int) ([]*domain.ProcessLog, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	
	buffer, exists := ps.logBuffers[processID]
	if !exists {
		// Try to load from storage
		logs, err := ps.storage.GetProcessLogs(processID, limit)
		if err != nil {
			return nil, fmt.Errorf("no logs found for process: %w", err)
		}
		return logs, nil
	}
	
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	
	// Return last N logs
	if limit <= 0 || limit > len(buffer.logs) {
		limit = len(buffer.logs)
	}
	
	start := len(buffer.logs) - limit
	if start < 0 {
		start = 0
	}
	
	result := make([]*domain.ProcessLog, limit)
	copy(result, buffer.logs[start:])
	
	return result, nil
}

// List lists processes with optional filter
func (ps *ProcessService) List(filter domain.ProcessFilter) ([]*domain.Process, error) {
	return ps.storage.ListProcesses(filter)
}

// Get retrieves a specific process
func (ps *ProcessService) Get(processID string) (*domain.Process, error) {
	return ps.storage.GetProcess(processID)
}

// Update updates process configuration
func (ps *ProcessService) Update(processID string, updates map[string]interface{}) (*domain.Process, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	process, err := ps.storage.GetProcess(processID)
	if err != nil {
		return nil, err
	}
	
	// Apply updates (simplified - in production would use reflection or specific update methods)
	if name, ok := updates["name"].(string); ok {
		process.Name = name
	}
	if env, ok := updates["environment"].(map[string]string); ok {
		process.Environment = env
	}
	if restart, ok := updates["restartPolicy"].(domain.RestartPolicy); ok {
		process.RestartPolicy = restart
	}
	
	process.UpdatedAt = time.Now()
	
	if err := ps.storage.SaveProcess(process.ProjectID, process); err != nil {
		return nil, err
	}
	
	return process, nil
}

// CreateGroup creates a new process group
func (ps *ProcessService) CreateGroup(group *domain.ProcessGroup) error {
	return ps.storage.SaveProcessGroup(group.ProjectID, group)
}

// AddToGroup adds a process to a group
func (ps *ProcessService) AddToGroup(groupID, processID string) error {
	// Implementation would update the group's process list
	return nil
}

// StartGroup starts all processes in a group
func (ps *ProcessService) StartGroup(groupID string) error {
	group, err := ps.storage.GetProcessGroup(groupID)
	if err != nil {
		return err
	}
	
	var lastErr error
	for _, processID := range group.ProcessIDs {
		if err := ps.Start(processID); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// StopGroup stops all processes in a group
func (ps *ProcessService) StopGroup(groupID string) error {
	group, err := ps.storage.GetProcessGroup(groupID)
	if err != nil {
		return err
	}
	
	var lastErr error
	for _, processID := range group.ProcessIDs {
		if err := ps.Stop(processID); err != nil {
			lastErr = err
		}
	}
	
	return lastErr
}

// captureOutput captures process output to logs
func (ps *ProcessService) captureOutput(processID string, pipe io.Reader, logType domain.LogType) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		ps.addLog(processID, logType, line)
	}
}

// addLog adds a log entry
func (ps *ProcessService) addLog(processID string, logType domain.LogType, message string) {
	log := domain.NewProcessLog(processID, logType, message)
	
	ps.mu.Lock()
	buffer, exists := ps.logBuffers[processID]
	ps.mu.Unlock()
	
	if !exists {
		return
	}
	
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	
	buffer.logs = append(buffer.logs, log)
	
	// Rotate if needed
	if len(buffer.logs) > buffer.maxSize {
		// Save older logs to storage
		toSave := buffer.logs[:buffer.maxSize/2]
		buffer.storage.SaveProcessLogs(toSave)
		
		// Keep newer logs in memory
		buffer.logs = buffer.logs[buffer.maxSize/2:]
	}
}

// healthCheckLoop monitors process health
func (ps *ProcessService) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ps.checkHealth()
		case <-ps.ctx.Done():
			return
		}
	}
}

// checkHealth checks health of all running processes
func (ps *ProcessService) checkHealth() {
	ps.mu.RLock()
	processes := make([]*ProcessInfo, 0, len(ps.processes))
	for _, info := range ps.processes {
		processes = append(processes, info)
	}
	ps.mu.RUnlock()
	
	for _, info := range processes {
		// Check if process is still running
		if info.Cmd.Process != nil {
			if err := info.Cmd.Process.Signal(syscall.Signal(0)); err != nil {
				// Process is not running
				ps.mu.Lock()
				info.Process.Status = domain.ProcessStatusCrashed
				now := time.Now()
				info.Process.StoppedAt = &now
				ps.storage.SaveProcess(info.Process.ProjectID, info.Process)
				delete(ps.processes, info.Process.ID)
				ps.mu.Unlock()
				
				ps.addLog(info.Process.ID, domain.LogTypeSystem, "Process health check failed - process not running")
			}
		}
		
		// Update last health check time
		now := time.Now()
		info.Process.LastHealthCheck = &now
		ps.storage.SaveProcess(info.Process.ProjectID, info.Process)
	}
}

// handleRestart handles process restart based on policy
func (ps *ProcessService) handleRestart(process *domain.Process) {
	if process.RestartPolicy.RetryCount >= process.RestartPolicy.MaxRetries {
		ps.addLog(process.ID, domain.LogTypeSystem, 
			fmt.Sprintf("Max restart attempts reached (%d/%d)", 
				process.RestartPolicy.RetryCount, process.RestartPolicy.MaxRetries))
		return
	}
	
	// Schedule restart after delay
	time.AfterFunc(process.RestartPolicy.RetryDelay, func() {
		process.RestartPolicy.RetryCount++
		now := time.Now()
		process.RestartPolicy.LastRestart = &now
		
		ps.addLog(process.ID, domain.LogTypeSystem, 
			fmt.Sprintf("Restarting process (attempt %d/%d)", 
				process.RestartPolicy.RetryCount, process.RestartPolicy.MaxRetries))
		
		if err := ps.Start(process.ID); err != nil {
			ps.addLog(process.ID, domain.LogTypeSystem, 
				fmt.Sprintf("Restart failed: %v", err))
		}
	})
}

// loadProcesses loads process state from storage on startup
func (ps *ProcessService) loadProcesses() {
	// This would reload any processes that should be running
	// For now, we'll keep it simple and not auto-restart on service startup
}

// Shutdown gracefully shuts down the process service
func (ps *ProcessService) Shutdown() {
	ps.cancel()
	
	// Stop all running processes
	ps.mu.Lock()
	for id := range ps.processes {
		ps.Stop(id)
	}
	ps.mu.Unlock()
	
	// Wait for all processes to stop
	timeout := time.After(30 * time.Second)
	for {
		ps.mu.RLock()
		count := len(ps.processes)
		ps.mu.RUnlock()
		
		if count == 0 {
			break
		}
		
		select {
		case <-timeout:
			// Force kill remaining processes
			ps.mu.Lock()
			for _, info := range ps.processes {
				if info.Cmd.Process != nil {
					info.Cmd.Process.Kill()
				}
			}
			ps.mu.Unlock()
			return
		case <-time.After(100 * time.Millisecond):
			// Check again
		}
	}
}