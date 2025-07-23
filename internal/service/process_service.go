package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	
	log.Printf("[DEBUG] Process creation started for: %s", process.Name)
	
	// Validate command
	if process.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	log.Printf("[DEBUG] Command validation passed")
	
	// Set defaults - use the directory where MCP server was started (user's project dir)
	if process.WorkingDir == "" {
		process.WorkingDir = ps.defaultWorkingDir
	}
	
	// Validate working directory exists
	log.Printf("[DEBUG] Checking working directory: %s", process.WorkingDir)
	if info, err := os.Stat(process.WorkingDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("working directory does not exist: %s", process.WorkingDir)
		}
		return fmt.Errorf("failed to check working directory: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("working directory is not a directory: %s", process.WorkingDir)
	}
	log.Printf("[DEBUG] Working directory validation passed")
	
	// Check if command is executable
	log.Printf("[DEBUG] Checking command executable: %s", process.Command)
	if !ps.isCommandExecutable(process.Command, process.WorkingDir) {
		return fmt.Errorf("command not found or not executable: %s. Make sure the command is in PATH or use an absolute path", process.Command)
	}
	log.Printf("[DEBUG] Command executable validation passed")
	
	// Validate environment variables
	log.Printf("[DEBUG] Validating environment variables")
	if err := ps.validateEnvironmentVariables(process.Environment); err != nil {
		return fmt.Errorf("invalid environment variables: %w", err)
	}
	log.Printf("[DEBUG] Environment variables validation passed")
	
	// Check for port conflicts if port is specified
	if process.Port > 0 {
		log.Printf("[DEBUG] Checking port availability: %d", process.Port)
		if err := ps.checkPortAvailable(process.Port); err != nil {
			return fmt.Errorf("port conflict: %w", err)
		}
		log.Printf("[DEBUG] Port availability check passed")
	}
	
	// Save to storage
	log.Printf("[DEBUG] Saving process to storage")
	if err := ps.storage.SaveProcess(process.ProjectID, process); err != nil {
		return fmt.Errorf("failed to save process: %w", err)
	}
	log.Printf("[DEBUG] Process saved to storage")
	
	// Initialize log buffer
	log.Printf("[DEBUG] Initializing log buffer")
	ps.logBuffers[process.ID] = &LogBuffer{
		logs:    make([]*domain.ProcessLog, 0, 1000),
		maxSize: 10000,
		storage: ps.storage,
	}
	
	log.Printf("[DEBUG] Process creation completed successfully for: %s", process.Name)
	return nil
}

// Start starts a process
func (ps *ProcessService) Start(processID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	// Get process from storage (handle partial IDs too)
	process, err := ps.getProcessForOperation(processID)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}
	
	// Use the full process ID for subsequent operations
	processID = process.ID
	
	if !process.CanStart() {
		return fmt.Errorf("process cannot be started in status: %s", process.Status)
	}
	
	// Validate working directory still exists
	if _, err := os.Stat(process.WorkingDir); err != nil {
		return fmt.Errorf("working directory no longer exists: %s", process.WorkingDir)
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
		// Clean up pipes on error
		stdout.Close()
		stderr.Close()
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
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ProcessService: Panic in process monitor for %s: %v", processID[:8], r)
				// Ensure cleanup happens even after panic
				ps.mu.Lock()
				delete(ps.processes, processID)
				ps.mu.Unlock()
			}
		}()
		
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
	buffer := ps.logBuffers[processID] // Safe since we hold the lock
	if buffer != nil {
		ps.addLogUnsafe(buffer, processID, domain.LogTypeSystem, fmt.Sprintf("Process started with PID %d", process.PID))
	}
	
	return nil
}

// Stop stops a running process
func (ps *ProcessService) Stop(processID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	
	// First check if process exists in storage (handle partial IDs too)
	process, err := ps.getProcessForOperation(processID)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}
	
	// Use the full process ID for subsequent operations
	processID = process.ID
	
	// Check if process is currently running in memory
	info, exists := ps.processes[processID]
	if !exists {
		// Process is not currently running - check its status and handle gracefully
		switch process.Status {
		case domain.ProcessStatusStopped:
			return nil // Already stopped - return success
		case domain.ProcessStatusFailed, domain.ProcessStatusCrashed:
			return nil // Already terminated - return success  
		case domain.ProcessStatusPending:
			return nil // Never started - return success
		default:
			// Process should be running but isn't in our map - mark as stopped
			now := time.Now()
			process.Status = domain.ProcessStatusStopped
			process.StoppedAt = &now
			process.UpdatedAt = now
			ps.storage.SaveProcess(process.ProjectID, process)
			return nil
		}
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
	buffer := ps.logBuffers[processID] // Safe since we hold the lock
	if buffer != nil {
		ps.addLogUnsafe(buffer, processID, domain.LogTypeSystem, "Process stop requested")
	}
	
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
	// Try direct lookup first
	process, err := ps.storage.GetProcess(processID)
	if err == nil {
		return process, nil
	}
	
	// If not found and ID looks truncated (less than 36 chars), try partial match
	if len(processID) < 36 {
		return ps.getProcessByPartialID(processID)
	}
	
	return nil, err
}

// getProcessForOperation gets process for operations like stop/start, handling partial IDs
func (ps *ProcessService) getProcessForOperation(processID string) (*domain.Process, error) {
	// Try direct lookup first
	process, err := ps.storage.GetProcess(processID)
	if err == nil {
		return process, nil
	}
	
	// If not found and ID looks truncated (less than 36 chars), try partial match
	if len(processID) < 36 {
		return ps.getProcessByPartialID(processID)
	}
	
	return nil, err
}

// getProcessByPartialID finds a process by partial ID match
func (ps *ProcessService) getProcessByPartialID(partialID string) (*domain.Process, error) {
	// Get all processes and find matches
	allProcesses, err := ps.storage.ListProcesses(domain.ProcessFilter{})
	if err != nil {
		return nil, err
	}
	
	var matches []*domain.Process
	for _, p := range allProcesses {
		if strings.HasPrefix(p.ID, partialID) {
			matches = append(matches, p)
		}
	}
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no process found with ID starting with %s", partialID)
	}
	
	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple processes found with ID starting with %s (found %d matches)", partialID, len(matches))
	}
	
	return matches[0], nil
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ProcessService: Panic in captureOutput for process %s: %v", processID[:8], r)
		}
	}()
	
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		ps.addLog(processID, logType, line)
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("ProcessService: Error reading %s for process %s: %v", logType, processID[:8], err)
	}
}

// addLog adds a log entry (thread-safe - acquires ps.mu lock)
func (ps *ProcessService) addLog(processID string, logType domain.LogType, message string) {
	ps.mu.Lock()
	buffer, exists := ps.logBuffers[processID]
	if exists {
		ps.addLogUnsafe(buffer, processID, logType, message)
	}
	ps.mu.Unlock()
}

// addLogUnsafe adds a log entry without acquiring ps.mu (for use within methods that already hold the lock)
func (ps *ProcessService) addLogUnsafe(buffer *LogBuffer, processID string, logType domain.LogType, message string) {
	log := domain.NewProcessLog(processID, logType, message)
	
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
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ProcessService: Panic in healthCheckLoop: %v", r)
			// Restart the health check loop after a panic
			go ps.healthCheckLoop()
		}
	}()
	
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
	
	// Copy necessary data to avoid race conditions
	processID := process.ID
	retryCount := process.RestartPolicy.RetryCount + 1
	maxRetries := process.RestartPolicy.MaxRetries
	
	// Schedule restart after delay
	time.AfterFunc(process.RestartPolicy.RetryDelay, func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("ProcessService: Panic in restart handler for %s: %v", processID[:8], r)
			}
		}()
		
		// Update restart policy atomically
		ps.mu.Lock()
		if p, err := ps.storage.GetProcess(processID); err == nil && p != nil {
			p.RestartPolicy.RetryCount = retryCount
			now := time.Now()
			p.RestartPolicy.LastRestart = &now
			ps.storage.SaveProcess(p.ProjectID, p)
		}
		ps.mu.Unlock()
		
		ps.addLog(processID, domain.LogTypeSystem, 
			fmt.Sprintf("Restarting process (attempt %d/%d)", retryCount, maxRetries))
		
		if err := ps.Start(processID); err != nil {
			ps.addLog(processID, domain.LogTypeSystem, 
				fmt.Sprintf("Restart failed: %v", err))
		}
	})
}

// loadProcesses loads process state from storage on startup and reconciles with system state
func (ps *ProcessService) loadProcesses() {
	log.Printf("ProcessService: Loading and reconciling process state...")
	
	// Get all processes from storage
	processes, err := ps.storage.ListProcesses(domain.ProcessFilter{})
	if err != nil {
		log.Printf("ProcessService: Failed to load processes from storage: %v", err)
		return
	}
	
	log.Printf("ProcessService: Found %d stored processes", len(processes))
	
	staleProceses := 0
	reconciledProcesses := 0
	
	for _, process := range processes {
		// Check if process was marked as running
		if process.Status == domain.ProcessStatusRunning {
			// Verify if the process is actually still running
			if ps.isProcessRunning(process) {
				log.Printf("ProcessService: Process %s (PID %d) is still running, will reconnect", 
					process.ID[:8], process.PID)
				// TODO: In a future enhancement, we could reconnect to running processes
				// For now, we'll mark them as stopped since we can't manage them
				ps.markProcessAsStopped(process)
				reconciledProcesses++
			} else {
				log.Printf("ProcessService: Process %s was marked running but is not active, marking as stopped", 
					process.ID[:8])
				ps.markProcessAsStopped(process)
				staleProceses++
			}
		}
	}
	
	log.Printf("ProcessService: Reconciliation complete - %d stale processes updated, %d processes reconciled", 
		staleProceses, reconciledProcesses)
}

// isProcessRunning checks if a process is actually running on the system
func (ps *ProcessService) isProcessRunning(process *domain.Process) bool {
	if process.PID == 0 {
		return false
	}
	
	// Try to send signal 0 to check if process exists
	// Signal 0 doesn't actually send a signal but checks if we can send to the process
	err := syscall.Kill(process.PID, 0)
	return err == nil
}

// markProcessAsStopped updates a process status to stopped
func (ps *ProcessService) markProcessAsStopped(process *domain.Process) {
	now := time.Now()
	process.Status = domain.ProcessStatusStopped
	process.StoppedAt = &now
	process.UpdatedAt = now
	
	// Save updated status to storage
	if err := ps.storage.SaveProcess(process.ProjectID, process); err != nil {
		log.Printf("ProcessService: Failed to update process %s status: %v", process.ID[:8], err)
	}
}

// isCommandExecutable checks if a command can be executed
func (ps *ProcessService) isCommandExecutable(command, workingDir string) bool {
	// Check if it's an absolute path
	if filepath.IsAbs(command) {
		info, err := os.Stat(command)
		if err != nil {
			return false
		}
		return info.Mode()&0111 != 0 // Check if executable
	}
	
	// Check if command exists in working directory
	localPath := filepath.Join(workingDir, command)
	if info, err := os.Stat(localPath); err == nil {
		return info.Mode()&0111 != 0
	}
	
	// Check if command exists in PATH
	_, err := exec.LookPath(command)
	return err == nil
}

// validateEnvironmentVariables validates environment variable names and values
func (ps *ProcessService) validateEnvironmentVariables(env map[string]string) error {
	for key, value := range env {
		// Check key format (must be valid environment variable name)
		if key == "" {
			return fmt.Errorf("environment variable name cannot be empty")
		}
		
		// Check for invalid characters in key
		for _, c := range key {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
				return fmt.Errorf("invalid character in environment variable name '%s': only letters, numbers, and underscores are allowed", key)
			}
		}
		
		// Check if key starts with a digit
		if key[0] >= '0' && key[0] <= '9' {
			return fmt.Errorf("environment variable name '%s' cannot start with a digit", key)
		}
		
		// Warn about potentially sensitive variables (don't block them, just warn)
		if ps.isSensitiveEnvVar(key) {
			log.Printf("ProcessService: Warning - setting potentially sensitive environment variable: %s", key)
		}
		
		// Check value length (reasonable limit)
		if len(value) > 32768 { // 32KB limit
			return fmt.Errorf("environment variable '%s' value too long (max 32KB)", key)
		}
	}
	return nil
}

// isSensitiveEnvVar checks if an environment variable might contain sensitive information
func (ps *ProcessService) isSensitiveEnvVar(key string) bool {
	sensitive := []string{
		"PASSWORD", "PASSWD", "SECRET", "KEY", "TOKEN", "API_KEY", "AUTH", 
		"CREDENTIAL", "PRIVATE", "CERT", "CERTIFICATE", "DB_PASSWORD",
	}
	
	upperKey := strings.ToUpper(key)
	for _, s := range sensitive {
		if strings.Contains(upperKey, s) {
			return true
		}
	}
	return false
}

// checkPortAvailable checks if a port is available for use
func (ps *ProcessService) checkPortAvailable(port int) error {
	// Check if port is already used by another managed process
	ps.mu.RLock()
	for _, info := range ps.processes {
		if info.Process.Port == port && (info.Process.Status == domain.ProcessStatusRunning || 
			info.Process.Status == domain.ProcessStatusStarting) {
			ps.mu.RUnlock()
			return fmt.Errorf("port %d is already in use by process '%s' (ID: %s)", 
				port, info.Process.Name, info.Process.ID[:8])
		}
	}
	ps.mu.RUnlock()
	
	// Check if port is available on the system
	if !ps.isPortAvailable(port) {
		// Suggest alternative ports
		alternatives := ps.suggestAlternativePorts(port)
		return fmt.Errorf("port %d is already in use by another process. Suggested alternatives: %v", 
			port, alternatives)
	}
	
	return nil
}

// isPortAvailable checks if a port is available on the system with timeout
func (ps *ProcessService) isPortAvailable(port int) bool {
	// Use a very short timeout to prevent hanging
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
	if err != nil {
		// If connection fails, port is available
		return true
	}
	conn.Close()
	// If connection succeeds, port is in use
	return false
}

// suggestAlternativePorts suggests alternative ports when there's a conflict
func (ps *ProcessService) suggestAlternativePorts(originalPort int) []int {
	alternatives := []int{}
	
	// Try only 3 immediate alternatives to prevent hanging
	for offset := 1; offset <= 3 && len(alternatives) < 3; offset++ {
		testPort := originalPort + offset
		if testPort <= 65535 && ps.isPortAvailable(testPort) {
			alternatives = append(alternatives, testPort)
		}
	}
	
	// If we didn't find enough nearby, add a few common alternatives without checking availability
	// to prevent hanging - let the user choose
	if len(alternatives) < 3 {
		commonAlts := map[int][]int{
			3000: {3001, 8000},
			8000: {8001, 3000},
			8080: {8081, 9080},
			5000: {5001, 8000},
		}
		
		if altPorts, exists := commonAlts[originalPort]; exists {
			for _, port := range altPorts {
				if len(alternatives) >= 3 {
					break
				}
				// Add without checking to prevent hanging - just suggest common alternatives
				alternatives = append(alternatives, port)
			}
		}
	}
	
	return alternatives
}

// Shutdown gracefully shuts down the process service
func (ps *ProcessService) Shutdown() {
	log.Printf("ProcessService: Starting shutdown...")
	ps.cancel()
	
	// Collect process IDs to stop (to avoid deadlock)
	ps.mu.Lock()
	processIDs := make([]string, 0, len(ps.processes))
	for id := range ps.processes {
		processIDs = append(processIDs, id)
	}
	processCount := len(processIDs)
	ps.mu.Unlock()
	
	// Stop all running processes (without holding the lock)
	log.Printf("ProcessService: Found %d processes to stop", processCount)
	for _, id := range processIDs {
		log.Printf("ProcessService: Stopping process %s", id)
		ps.Stop(id)
	}
	
	// Wait for all processes to stop
	timeout := time.After(30 * time.Second)
	for {
		ps.mu.RLock()
		count := len(ps.processes)
		ps.mu.RUnlock()
		
		if count == 0 {
			log.Printf("ProcessService: All processes stopped successfully")
			break
		}
		
		log.Printf("ProcessService: Waiting for %d processes to stop...", count)
		
		select {
		case <-timeout:
			// Force kill remaining processes
			log.Printf("ProcessService: Timeout reached, force killing remaining processes")
			ps.mu.Lock()
			for _, info := range ps.processes {
				if info.Cmd.Process != nil {
					log.Printf("ProcessService: Force killing process PID %d", info.Cmd.Process.Pid)
					info.Cmd.Process.Kill()
				}
			}
			ps.mu.Unlock()
			return
		case <-time.After(100 * time.Millisecond):
			// Check again
		}
	}
	log.Printf("ProcessService: Shutdown completed")
}