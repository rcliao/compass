package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rcliao/compass/internal/domain"
)

// =============================================================================
// PROCESS ORCHESTRATOR - Coordinates all components (REPLACES ProcessService)
// =============================================================================

// ProcessOrchestrator manages all process operations using the actor model
type ProcessOrchestrator struct {
	// Core components
	stateManager *StateManager
	logPipeline  *LogPipeline
	
	// Storage
	storage ProcessStorage
	
	// Communication channels
	logsCh   chan LogEntry
	eventsCh chan ProcessEvent
	
	// Configuration
	defaultWorkingDir string
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	
	// Statistics
	totalProcessesCreated int64
	totalCommandsHandled  int64
	
	// Health monitoring
	lastHealthCheck time.Time
	isHealthy       atomic.Bool
}

// ProcessOrchestratorConfig holds configuration
type ProcessOrchestratorConfig struct {
	DefaultWorkingDir string
	LogPipelineConfig LogPipelineConfig
}

// DefaultProcessOrchestratorConfig returns default configuration
func DefaultProcessOrchestratorConfig() ProcessOrchestratorConfig {
	return ProcessOrchestratorConfig{
		DefaultWorkingDir: ".",
		LogPipelineConfig: DefaultLogPipelineConfig(),
	}
}

// NewProcessOrchestrator creates a new process orchestrator
func NewProcessOrchestrator(storage ProcessStorage, config ProcessOrchestratorConfig) *ProcessOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create communication channels
	logsCh := make(chan LogEntry, 1000)
	eventsCh := make(chan ProcessEvent, 1000)
	
	// Create components
	stateManager := NewStateManager(storage)
	logPipeline := NewLogPipeline(storage, config.LogPipelineConfig)
	
	orchestrator := &ProcessOrchestrator{
		stateManager:      stateManager,
		logPipeline:       logPipeline,
		storage:           storage,
		logsCh:            logsCh,
		eventsCh:          eventsCh,
		defaultWorkingDir: config.DefaultWorkingDir,
		ctx:               ctx,
		cancel:            cancel,
		done:              make(chan struct{}),
		lastHealthCheck:   time.Now(),
	}
	
	orchestrator.isHealthy.Store(true)
	
	return orchestrator
}

// Initialize starts all components
func (po *ProcessOrchestrator) Initialize() error {
	log.Println("ProcessOrchestrator: Starting...")
	
	// Start core components
	po.stateManager.Start()
	po.logPipeline.Start()
	
	// Start orchestrator's main loop
	go po.run()
	
	// Start health monitoring
	go po.healthMonitor()
	
	log.Println("ProcessOrchestrator: Started successfully")
	return nil
}

// Create creates a new process definition
func (po *ProcessOrchestrator) Create(process *domain.Process) error {
	atomic.AddInt64(&po.totalCommandsHandled, 1)
	
	log.Printf("ProcessOrchestrator: Creating process %s (%s)", process.ID[:8], process.Name)
	
	// Validate process definition
	if err := po.validateProcess(process); err != nil {
		return fmt.Errorf("process validation failed: %w", err)
	}
	
	// Set defaults
	if process.WorkingDir == "" {
		process.WorkingDir = po.defaultWorkingDir
	}
	
	// Generate ID if not provided
	if process.ID == "" {
		process.ID = uuid.New().String()
	}
	
	// Set initial status
	process.Status = domain.ProcessStatusPending
	process.CreatedAt = time.Now()
	process.UpdatedAt = time.Now()
	
	// Check port availability if specified
	if process.Port > 0 {
		if err := po.checkPortAvailable(process.Port, process.ID); err != nil {
			return fmt.Errorf("port conflict: %w", err)
		}
	}
	
	// Create actor (but don't start the process yet)
	actor := NewProcessActor(process, po.logsCh, po.eventsCh)
	actor.Start()
	
	// Register with state manager
	po.stateManager.RegisterProcess(process, actor)
	
	atomic.AddInt64(&po.totalProcessesCreated, 1)
	log.Printf("ProcessOrchestrator: Created process %s successfully", process.ID[:8])
	
	return nil
}

// Start starts a process
func (po *ProcessOrchestrator) Start(processID string) error {
	atomic.AddInt64(&po.totalCommandsHandled, 1)
	
	log.Printf("ProcessOrchestrator: Starting process %s", processID[:8])
	
	// Get actor from state manager
	actor, err := po.stateManager.GetProcessActor(processID)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}
	
	// Send start command to actor
	responseCh := make(chan ProcessResponse, 1)
	cmd := ProcessCommand{
		Type:       "start",
		ResponseCh: responseCh,
		Timeout:    10 * time.Second,
	}
	
	if err := actor.SendCommand(cmd); err != nil {
		return fmt.Errorf("failed to send start command: %w", err)
	}
	
	// Wait for response
	select {
	case response := <-responseCh:
		if !response.Success {
			return fmt.Errorf("start failed: %w", response.Error)
		}
		log.Printf("ProcessOrchestrator: Started process %s successfully", processID[:8])
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("start command timeout")
	}
}

// Stop stops a process
func (po *ProcessOrchestrator) Stop(processID string) error {
	atomic.AddInt64(&po.totalCommandsHandled, 1)
	
	log.Printf("ProcessOrchestrator: Stopping process %s", processID[:8])
	
	// Get actor from state manager
	actor, err := po.stateManager.GetProcessActor(processID)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}
	
	// Send stop command to actor
	responseCh := make(chan ProcessResponse, 1)
	cmd := ProcessCommand{
		Type:       "stop",
		ResponseCh: responseCh,
		Timeout:    10 * time.Second,
	}
	
	if err := actor.SendCommand(cmd); err != nil {
		return fmt.Errorf("failed to send stop command: %w", err)
	}
	
	// Wait for response
	select {
	case response := <-responseCh:
		if !response.Success {
			return fmt.Errorf("stop failed: %w", response.Error)
		}
		log.Printf("ProcessOrchestrator: Stopped process %s successfully", processID[:8])
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("stop command timeout")
	}
}

// Get retrieves a process
func (po *ProcessOrchestrator) Get(processID string) (*domain.Process, error) {
	atomic.AddInt64(&po.totalCommandsHandled, 1)
	
	// Handle partial IDs
	if len(processID) < 36 {
		fullID, err := po.resolvePartialID(processID)
		if err != nil {
			return nil, err
		}
		processID = fullID
	}
	
	return po.stateManager.GetProcess(processID)
}

// List lists processes with optional filtering
func (po *ProcessOrchestrator) List(filter domain.ProcessFilter) ([]*domain.Process, error) {
	atomic.AddInt64(&po.totalCommandsHandled, 1)
	
	return po.stateManager.ListProcesses(filter)
}

// GetLogs retrieves logs for a process
func (po *ProcessOrchestrator) GetLogs(processID string, limit int) ([]*domain.ProcessLog, error) {
	atomic.AddInt64(&po.totalCommandsHandled, 1)
	
	// Handle partial IDs
	if len(processID) < 36 {
		fullID, err := po.resolvePartialID(processID)
		if err != nil {
			return nil, err
		}
		processID = fullID
	}
	
	return po.logPipeline.GetLogs(processID, limit)
}

// Update updates process configuration
func (po *ProcessOrchestrator) Update(processID string, updates map[string]interface{}) (*domain.Process, error) {
	atomic.AddInt64(&po.totalCommandsHandled, 1)
	
	// Get current process
	process, err := po.Get(processID)
	if err != nil {
		return nil, err
	}
	
	// Apply updates
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
	
	// Save to storage
	if err := po.storage.SaveProcess(process.ProjectID, process); err != nil {
		return nil, err
	}
	
	return process, nil
}

// CreateGroup creates a process group
func (po *ProcessOrchestrator) CreateGroup(group *domain.ProcessGroup) error {
	return po.storage.SaveProcessGroup(group.ProjectID, group)
}

// StartGroup starts all processes in a group
func (po *ProcessOrchestrator) StartGroup(groupID string) error {
	group, err := po.storage.GetProcessGroup(groupID)
	if err != nil {
		return err
	}
	
	var lastErr error
	for _, processID := range group.ProcessIDs {
		if err := po.Start(processID); err != nil {
			log.Printf("ProcessOrchestrator: Failed to start process %s in group: %v", processID[:8], err)
			lastErr = err
		}
	}
	
	return lastErr
}

// StopGroup stops all processes in a group
func (po *ProcessOrchestrator) StopGroup(groupID string) error {
	group, err := po.storage.GetProcessGroup(groupID)
	if err != nil {
		return err
	}
	
	var lastErr error
	for _, processID := range group.ProcessIDs {
		if err := po.Stop(processID); err != nil {
			log.Printf("ProcessOrchestrator: Failed to stop process %s in group: %v", processID[:8], err)
			lastErr = err
		}
	}
	
	return lastErr
}

// GetStatistics returns orchestrator statistics
func (po *ProcessOrchestrator) GetStatistics() map[string]interface{} {
	stats := map[string]interface{}{
		"total_processes_created": atomic.LoadInt64(&po.totalProcessesCreated),
		"total_commands_handled":  atomic.LoadInt64(&po.totalCommandsHandled),
		"is_healthy":              po.isHealthy.Load(),
		"last_health_check":       po.lastHealthCheck,
	}
	
	// Add component statistics
	stats["state_manager"] = po.stateManager.GetStatistics()
	stats["log_pipeline"] = po.logPipeline.GetStatistics()
	
	return stats
}

// validateProcess validates a process definition
func (po *ProcessOrchestrator) validateProcess(process *domain.Process) error {
	if process.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	
	if process.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	
	// Validate working directory
	if process.WorkingDir != "" {
		// In a real implementation, you'd check if the directory exists
		// For now, we'll just check it's not empty
	}
	
	// Validate port
	if process.Port < 0 || process.Port > 65535 {
		return fmt.Errorf("invalid port: %d", process.Port)
	}
	
	// Validate environment variables
	for key, _ := range process.Environment {
		if key == "" {
			return fmt.Errorf("environment variable name cannot be empty")
		}
	}
	
	return nil
}

// checkPortAvailable checks if a port is available
func (po *ProcessOrchestrator) checkPortAvailable(port int, excludeProcessID string) error {
	// Check if port is used by another managed process
	filter := domain.ProcessFilter{
		Status: &[]domain.ProcessStatus{domain.ProcessStatusRunning, domain.ProcessStatusStarting}[0],
	}
	
	processes, err := po.stateManager.ListProcesses(filter)
	if err != nil {
		return fmt.Errorf("failed to check existing processes: %w", err)
	}
	
	for _, process := range processes {
		if process.Port == port && process.ID != excludeProcessID {
			return fmt.Errorf("port %d is already in use by process '%s' (ID: %s)", 
				port, process.Name, process.ID[:8])
		}
	}
	
	// Check if port is available on the system
	if !po.isPortAvailable(port) {
		return fmt.Errorf("port %d is already in use by another process", port)
	}
	
	return nil
}

// isPortAvailable checks if a port is available on the system
func (po *ProcessOrchestrator) isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// resolvePartialID resolves a partial process ID to a full ID
func (po *ProcessOrchestrator) resolvePartialID(partialID string) (string, error) {
	processes, err := po.stateManager.ListProcesses(domain.ProcessFilter{})
	if err != nil {
		return "", err
	}
	
	var matches []string
	for _, process := range processes {
		if strings.HasPrefix(process.ID, partialID) {
			matches = append(matches, process.ID)
		}
	}
	
	if len(matches) == 0 {
		return "", fmt.Errorf("no process found with ID starting with %s", partialID)
	}
	
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple processes found with ID starting with %s (found %d matches)", 
			partialID, len(matches))
	}
	
	return matches[0], nil
}

// run is the main orchestrator loop
func (po *ProcessOrchestrator) run() {
	defer close(po.done)
	
	log.Println("ProcessOrchestrator: Main loop started")
	
	for {
		select {
		case event := <-po.eventsCh:
			// Forward events to state manager
			po.stateManager.HandleEvent(event)
			
		case logEntry := <-po.logsCh:
			// Forward logs to log pipeline
			po.logPipeline.SendLog(logEntry)
			
		case <-po.ctx.Done():
			log.Println("ProcessOrchestrator: Main loop stopping")
			return
		}
	}
}

// healthMonitor monitors the health of all components
func (po *ProcessOrchestrator) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			po.performHealthCheck()
			
		case <-po.ctx.Done():
			return
		}
	}
}

// performHealthCheck checks the health of all components
func (po *ProcessOrchestrator) performHealthCheck() {
	po.lastHealthCheck = time.Now()
	
	healthy := true
	
	// Check state manager health
	if !po.stateManager.IsHealthy() {
		log.Println("ProcessOrchestrator: StateManager is unhealthy")
		healthy = false
	}
	
	// Check log pipeline health
	if !po.logPipeline.IsHealthy() {
		log.Println("ProcessOrchestrator: LogPipeline is unhealthy")
		healthy = false
	}
	
	// Check channel usage
	logChannelUsage := float64(len(po.logsCh)) / 1000.0
	eventChannelUsage := float64(len(po.eventsCh)) / 1000.0
	
	if logChannelUsage > 0.8 {
		log.Printf("ProcessOrchestrator: Log channel usage high: %.1f%%", logChannelUsage*100)
		healthy = false
	}
	
	if eventChannelUsage > 0.8 {
		log.Printf("ProcessOrchestrator: Event channel usage high: %.1f%%", eventChannelUsage*100)
		healthy = false
	}
	
	po.isHealthy.Store(healthy)
	
	if healthy {
		log.Println("ProcessOrchestrator: Health check passed")
	} else {
		log.Println("ProcessOrchestrator: Health check failed")
	}
}

// Shutdown gracefully shuts down the orchestrator
func (po *ProcessOrchestrator) Shutdown() {
	log.Println("ProcessOrchestrator: Starting shutdown...")
	
	// Stop all running processes
	processes, err := po.stateManager.GetRunningProcesses()
	if err == nil {
		for _, process := range processes {
			log.Printf("ProcessOrchestrator: Stopping process %s during shutdown", process.ID[:8])
			po.Stop(process.ID)
		}
	}
	
	// Cancel context to stop all goroutines
	po.cancel()
	
	// Wait for main loop to finish
	select {
	case <-po.done:
		log.Println("ProcessOrchestrator: Main loop stopped")
	case <-time.After(10 * time.Second):
		log.Println("ProcessOrchestrator: Main loop stop timeout")
	}
	
	// Stop components
	po.stateManager.Stop()
	po.logPipeline.Stop()
	
	log.Println("ProcessOrchestrator: Shutdown complete")
}

// IsHealthy returns the overall health status
func (po *ProcessOrchestrator) IsHealthy() bool {
	return po.isHealthy.Load()
}