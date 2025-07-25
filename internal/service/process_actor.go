package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rcliao/compass/internal/domain"
)

// =============================================================================
// PROCESS ACTOR - Each process gets its own goroutine (NO MUTEXES!)
// =============================================================================

// ProcessCommand represents commands sent to process actors
type ProcessCommand struct {
	Type       string
	ResponseCh chan ProcessResponse
	Data       interface{}
	Timeout    time.Duration
}

// ProcessResponse represents responses from process actors
type ProcessResponse struct {
	Success bool
	Data    interface{}
	Error   error
}

// ProcessEvent represents events from process actors
type ProcessEvent struct {
	ProcessID string
	Type      string
	Data      interface{}
	Timestamp time.Time
}

// LogEntry represents a single log entry
type LogEntry struct {
	ProcessID string
	Type      domain.LogType
	Message   string
	Timestamp time.Time
}

// ProcessActor manages a single process lifecycle - NO SHARED STATE!
type ProcessActor struct {
	// Immutable data
	id      string
	process *domain.Process
	
	// Actor state (owned by this goroutine only)
	cmd         *exec.Cmd
	ctx         context.Context
	cancel      context.CancelFunc
	running     atomic.Bool
	
	// Communication channels
	commandCh chan ProcessCommand
	logsCh    chan LogEntry
	eventsCh  chan ProcessEvent
	done      chan struct{}
	
	// Internal state
	startTime   time.Time
	stopTime    time.Time
	exitCode    int
	lastError   error
}

// NewProcessActor creates a new process actor
func NewProcessActor(process *domain.Process, logsCh chan LogEntry, eventsCh chan ProcessEvent) *ProcessActor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ProcessActor{
		id:        process.ID,
		process:   process,
		ctx:       ctx,
		cancel:    cancel,
		commandCh: make(chan ProcessCommand, 10), // Buffered to prevent blocking
		logsCh:    logsCh,
		eventsCh:  eventsCh,
		done:      make(chan struct{}),
	}
}

// Start begins the actor's main loop
func (pa *ProcessActor) Start() {
	go pa.run()
}

// SendCommand sends a command to the actor (non-blocking with timeout)
func (pa *ProcessActor) SendCommand(cmd ProcessCommand) error {
	// Set default timeout if not specified
	if cmd.Timeout == 0 {
		cmd.Timeout = 5 * time.Second
	}
	
	select {
	case pa.commandCh <- cmd:
		return nil
	case <-time.After(cmd.Timeout):
		// Prevent hanging - always respond to caller
		if cmd.ResponseCh != nil {
			select {
			case cmd.ResponseCh <- ProcessResponse{
				Success: false,
				Error:   fmt.Errorf("actor command timeout after %v", cmd.Timeout),
			}:
			default:
			}
		}
		return fmt.Errorf("command send timeout")
	}
}

// Main actor loop - handles all process operations
func (pa *ProcessActor) run() {
	defer close(pa.done)
	defer pa.sendEvent("actor_stopped", nil)
	
	pa.sendLog(domain.LogTypeSystem, fmt.Sprintf("Process actor started for %s", pa.process.Name))
	
	for {
		select {
		case cmd := <-pa.commandCh:
			pa.handleCommand(cmd)
			
		case <-pa.ctx.Done():
			pa.sendLog(domain.LogTypeSystem, "Process actor shutting down")
			if pa.running.Load() {
				pa.forceStop()
			}
			return
		}
	}
}

// handleCommand processes incoming commands
func (pa *ProcessActor) handleCommand(cmd ProcessCommand) {
	var response ProcessResponse
	
	switch cmd.Type {
	case "start":
		response = pa.startProcess()
	case "stop":
		response = pa.stopProcess()
	case "kill":
		response = pa.killProcess()
	case "status":
		response = pa.getStatus()
	case "restart":
		response = pa.restartProcess()
	default:
		response = ProcessResponse{
			Success: false,
			Error:   fmt.Errorf("unknown command: %s", cmd.Type),
		}
	}
	
	// Always respond (prevents hanging)
	if cmd.ResponseCh != nil {
		select {
		case cmd.ResponseCh <- response:
		case <-time.After(1 * time.Second):
			// Prevent deadlock if receiver is gone
			pa.sendLog(domain.LogTypeSystem, "Command response timeout - receiver may be gone")
		}
	}
}

// startProcess starts the underlying process
func (pa *ProcessActor) startProcess() ProcessResponse {
	if pa.running.Load() {
		return ProcessResponse{
			Success: false,
			Error:   fmt.Errorf("process already running"),
		}
	}
	
	pa.sendLog(domain.LogTypeSystem, "Starting process...")
	
	// Create command with timeout context
	pa.cmd = exec.CommandContext(pa.ctx, pa.process.Command, pa.process.Args...)
	pa.cmd.Dir = pa.process.WorkingDir
	
	// Set environment variables
	if len(pa.process.Environment) > 0 {
		env := pa.cmd.Environ()
		for k, v := range pa.process.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		pa.cmd.Env = env
	}
	
	// Force unbuffered output for real-time log capture
	if pa.cmd.Env == nil {
		pa.cmd.Env = pa.cmd.Environ()
	}
	pa.cmd.Env = append(pa.cmd.Env, "PYTHONUNBUFFERED=1", "PYTHONIOENCODING=utf-8")
	
	// Setup pipes for output capture
	stdout, err := pa.cmd.StdoutPipe()
	if err != nil {
		return ProcessResponse{Success: false, Error: fmt.Errorf("failed to create stdout pipe: %w", err)}
	}
	
	stderr, err := pa.cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		return ProcessResponse{Success: false, Error: fmt.Errorf("failed to create stderr pipe: %w", err)}
	}
	
	// Set stdin to nil to prevent process from waiting for input
	pa.cmd.Stdin = nil
	
	// Start the process
	if err := pa.cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		return ProcessResponse{Success: false, Error: fmt.Errorf("failed to start process: %w", err)}
	}
	
	// Update actor state
	pa.running.Store(true)
	pa.startTime = time.Now()
	pa.process.PID = pa.cmd.Process.Pid
	pa.process.Status = domain.ProcessStatusRunning
	
	// Start log capture routines (non-blocking)
	go pa.captureOutput(stdout, domain.LogTypeStdout)
	go pa.captureOutput(stderr, domain.LogTypeStderr)
	
	// Monitor process completion (non-blocking)
	go pa.monitorProcess()
	
	pa.sendEvent("process_started", map[string]interface{}{
		"pid":        pa.process.PID,
		"start_time": pa.startTime,
	})
	
	pa.sendLog(domain.LogTypeSystem, fmt.Sprintf("Process started with PID %d", pa.process.PID))
	
	return ProcessResponse{
		Success: true,
		Data: map[string]interface{}{
			"pid":        pa.process.PID,
			"start_time": pa.startTime,
		},
	}
}

// stopProcess gracefully stops the process
func (pa *ProcessActor) stopProcess() ProcessResponse {
	if !pa.running.Load() {
		return ProcessResponse{
			Success: false,
			Error:   fmt.Errorf("process not running"),
		}
	}
	
	pa.sendLog(domain.LogTypeSystem, "Stopping process gracefully...")
	
	if pa.cmd != nil && pa.cmd.Process != nil {
		// Send SIGTERM for graceful shutdown
		if err := pa.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			pa.sendLog(domain.LogTypeSystem, fmt.Sprintf("SIGTERM failed: %v, trying SIGKILL", err))
			// If SIGTERM fails, force kill
			if err := pa.cmd.Process.Kill(); err != nil {
				return ProcessResponse{Success: false, Error: fmt.Errorf("failed to kill process: %w", err)}
			}
		} else {
			// Start a timer for forced kill if graceful shutdown takes too long
			go func() {
				time.Sleep(5 * time.Second)
				if pa.running.Load() {
					pa.sendLog(domain.LogTypeSystem, "Graceful shutdown timeout, sending SIGKILL")
					if pa.cmd != nil && pa.cmd.Process != nil {
						pa.cmd.Process.Kill()
					}
				}
			}()
		}
	}
	
	return ProcessResponse{Success: true}
}

// killProcess forcefully kills the process
func (pa *ProcessActor) killProcess() ProcessResponse {
	if !pa.running.Load() {
		return ProcessResponse{
			Success: false,
			Error:   fmt.Errorf("process not running"),
		}
	}
	
	pa.sendLog(domain.LogTypeSystem, "Force killing process...")
	
	if pa.cmd != nil && pa.cmd.Process != nil {
		if err := pa.cmd.Process.Kill(); err != nil {
			return ProcessResponse{Success: false, Error: fmt.Errorf("failed to kill process: %w", err)}
		}
	}
	
	return ProcessResponse{Success: true}
}

// restartProcess stops and starts the process
func (pa *ProcessActor) restartProcess() ProcessResponse {
	if pa.running.Load() {
		stopResp := pa.stopProcess()
		if !stopResp.Success {
			return stopResp
		}
		
		// Wait for process to stop (with timeout)
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-timeout:
				pa.forceStop()
				break
			case <-ticker.C:
				if !pa.running.Load() {
					goto startProcess
				}
			}
		}
	}
	
startProcess:
	// Small delay to ensure cleanup is complete
	time.Sleep(100 * time.Millisecond)
	return pa.startProcess()
}

// getStatus returns current process status
func (pa *ProcessActor) getStatus() ProcessResponse {
	status := map[string]interface{}{
		"id":          pa.id,
		"name":        pa.process.Name,
		"command":     pa.process.Command,
		"args":        pa.process.Args,
		"working_dir": pa.process.WorkingDir,
		"status":      pa.process.Status,
		"pid":         pa.process.PID,
		"running":     pa.running.Load(),
		"start_time":  pa.startTime,
		"stop_time":   pa.stopTime,
		"exit_code":   pa.exitCode,
	}
	
	if pa.lastError != nil {
		status["last_error"] = pa.lastError.Error()
	}
	
	return ProcessResponse{
		Success: true,
		Data:    status,
	}
}

// captureOutput captures process output and sends to log pipeline
func (pa *ProcessActor) captureOutput(pipe io.Reader, logType domain.LogType) {
	defer func() {
		if r := recover(); r != nil {
			pa.sendLog(domain.LogTypeSystem, fmt.Sprintf("Panic in output capture (%s): %v", logType, r))
		}
	}()
	
	scanner := bufio.NewScanner(pipe)
	lineCount := 0
	
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			pa.sendLog(logType, line)
			lineCount++
			
			// Periodic debug info for active processes
			if lineCount%100 == 0 {
				pa.sendLog(domain.LogTypeSystem, 
					fmt.Sprintf("Captured %d lines from %s", lineCount, logType))
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		pa.sendLog(domain.LogTypeSystem, 
			fmt.Sprintf("Output capture error (%s): %v", logType, err))
	}
	
	pa.sendLog(domain.LogTypeSystem, 
		fmt.Sprintf("Output capture finished (%s): %d lines total", logType, lineCount))
}

// monitorProcess waits for process completion and updates state
func (pa *ProcessActor) monitorProcess() {
	if pa.cmd == nil {
		return
	}
	
	// Wait for process to complete
	err := pa.cmd.Wait()
	
	// Update state
	pa.running.Store(false)
	pa.stopTime = time.Now()
	pa.lastError = err
	
	if pa.cmd.ProcessState != nil {
		pa.exitCode = pa.cmd.ProcessState.ExitCode()
	}
	
	// Determine final status
	if err != nil {
		if pa.exitCode == -1 {
			pa.process.Status = domain.ProcessStatusCrashed
		} else {
			pa.process.Status = domain.ProcessStatusFailed
		}
	} else {
		pa.process.Status = domain.ProcessStatusStopped
	}
	
	pa.sendEvent("process_stopped", map[string]interface{}{
		"exit_code": pa.exitCode,
		"stop_time": pa.stopTime,
		"status":    pa.process.Status,
		"error":     err,
	})
	
	pa.sendLog(domain.LogTypeSystem, 
		fmt.Sprintf("Process exited with code %d, status: %s", pa.exitCode, pa.process.Status))
}

// forceStop forcefully stops everything (used during shutdown)
func (pa *ProcessActor) forceStop() {
	if pa.cmd != nil && pa.cmd.Process != nil {
		pa.cmd.Process.Kill()
	}
	pa.running.Store(false)
	pa.stopTime = time.Now()
	pa.process.Status = domain.ProcessStatusStopped
}

// sendLog sends a log entry to the log pipeline (never blocks)
func (pa *ProcessActor) sendLog(logType domain.LogType, message string) {
	entry := LogEntry{
		ProcessID: pa.id,
		Type:      logType,
		Message:   message,
		Timestamp: time.Now(),
	}
	
	select {
	case pa.logsCh <- entry:
		// Log sent successfully
	default:
		// Log pipeline is full - drop the log to prevent blocking
		// In production, we might want to increment a "dropped logs" counter
		log.Printf("LOG DROPPED for process %s: %s", pa.id[:8], message)
	}
}

// sendEvent sends an event to the event pipeline (never blocks)
func (pa *ProcessActor) sendEvent(eventType string, data interface{}) {
	event := ProcessEvent{
		ProcessID: pa.id,
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}
	
	select {
	case pa.eventsCh <- event:
		// Event sent successfully
	default:
		// Event pipeline is full - drop the event to prevent blocking
		log.Printf("EVENT DROPPED for process %s: %s", pa.id[:8], eventType)
	}
}

// Stop gracefully stops the actor
func (pa *ProcessActor) Stop() {
	pa.cancel()
	
	// Wait for actor to finish with timeout
	select {
	case <-pa.done:
		// Clean shutdown
	case <-time.After(10 * time.Second):
		// Force shutdown if it takes too long
		log.Printf("ProcessActor %s shutdown timeout", pa.id[:8])
	}
}

// IsRunning returns whether the process is currently running
func (pa *ProcessActor) IsRunning() bool {
	return pa.running.Load()
}

// GetID returns the process ID
func (pa *ProcessActor) GetID() string {
	return pa.id
}