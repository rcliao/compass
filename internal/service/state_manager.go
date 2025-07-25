package service

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/rcliao/compass/internal/domain"
)

// =============================================================================
// STATE MANAGER - Centralized process state (NO MUTEXES - single writer!)
// =============================================================================

// ProcessState represents the current state of a process
type ProcessState struct {
	Process   *domain.Process
	Actor     *ProcessActor
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StateQuery represents a query for process state
type StateQuery struct {
	Type       string
	Filter     interface{}
	ResponseCh chan StateQueryResponse
}

// StateQueryResponse represents the response to a state query
type StateQueryResponse struct {
	Success bool
	Data    interface{}
	Error   error
}

// StateUpdate represents an update to process state
type StateUpdate struct {
	Type      string
	ProcessID string
	Data      interface{}
	Timestamp time.Time
}

// StateManager manages all process state centrally (single writer pattern)
type StateManager struct {
	// Process state storage (no mutex needed - single writer)
	processes map[string]*ProcessState
	
	// Communication channels
	queryCh   chan StateQuery
	updateCh  chan StateUpdate
	eventsCh  chan ProcessEvent
	
	// Control
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	
	// Statistics
	totalProcesses    int64
	totalQueries      int64
	totalUpdates      int64
	totalEvents       int64
	
	// Storage interface
	storage ProcessStorage
}

// NewStateManager creates a new state manager
func NewStateManager(storage ProcessStorage) *StateManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &StateManager{
		processes: make(map[string]*ProcessState),
		queryCh:   make(chan StateQuery, 100),
		updateCh:  make(chan StateUpdate, 100),
		eventsCh:  make(chan ProcessEvent, 1000),
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		storage:   storage,
	}
}

// Start begins the state manager
func (sm *StateManager) Start() {
	go sm.run()
}

// RegisterProcess registers a new process and its actor
func (sm *StateManager) RegisterProcess(process *domain.Process, actor *ProcessActor) {
	update := StateUpdate{
		Type:      "register",
		ProcessID: process.ID,
		Data: map[string]interface{}{
			"process": process,
			"actor":   actor,
		},
		Timestamp: time.Now(),
	}
	
	select {
	case sm.updateCh <- update:
	case <-time.After(5 * time.Second):
		log.Printf("StateManager: Failed to register process %s (timeout)", process.ID[:8])
	}
}

// UnregisterProcess removes a process from state
func (sm *StateManager) UnregisterProcess(processID string) {
	update := StateUpdate{
		Type:      "unregister",
		ProcessID: processID,
		Timestamp: time.Now(),
	}
	
	select {
	case sm.updateCh <- update:
	case <-time.After(5 * time.Second):
		log.Printf("StateManager: Failed to unregister process %s (timeout)", processID[:8])
	}
}

// HandleEvent processes events from process actors
func (sm *StateManager) HandleEvent(event ProcessEvent) {
	select {
	case sm.eventsCh <- event:
	default:
		// Drop event if channel is full
		log.Printf("StateManager: Dropped event %s for process %s", event.Type, event.ProcessID[:8])
	}
}

// Query performs a synchronous query on process state
func (sm *StateManager) Query(queryType string, filter interface{}) (interface{}, error) {
	responseCh := make(chan StateQueryResponse, 1)
	
	query := StateQuery{
		Type:       queryType,
		Filter:     filter,
		ResponseCh: responseCh,
	}
	
	select {
	case sm.queryCh <- query:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("query timeout")
	}
	
	select {
	case response := <-responseCh:
		if response.Success {
			return response.Data, nil
		}
		return nil, response.Error
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("query response timeout")
	}
}

// GetProcess retrieves a single process
func (sm *StateManager) GetProcess(processID string) (*domain.Process, error) {
	data, err := sm.Query("get_process", processID)
	if err != nil {
		return nil, err
	}
	
	process, ok := data.(*domain.Process)
	if !ok {
		return nil, fmt.Errorf("invalid process data returned")
	}
	
	return process, nil
}

// ListProcesses retrieves all processes matching filter
func (sm *StateManager) ListProcesses(filter domain.ProcessFilter) ([]*domain.Process, error) {
	data, err := sm.Query("list_processes", filter)
	if err != nil {
		return nil, err
	}
	
	processes, ok := data.([]*domain.Process)
	if !ok {
		return nil, fmt.Errorf("invalid process list returned")
	}
	
	return processes, nil
}

// GetProcessActor retrieves the actor for a process
func (sm *StateManager) GetProcessActor(processID string) (*ProcessActor, error) {
	data, err := sm.Query("get_actor", processID)
	if err != nil {
		return nil, err
	}
	
	actor, ok := data.(*ProcessActor)
	if !ok {
		return nil, fmt.Errorf("invalid actor data returned")
	}
	
	return actor, nil
}

// GetRunningProcesses retrieves all currently running processes
func (sm *StateManager) GetRunningProcesses() ([]*domain.Process, error) {
	filter := domain.ProcessFilter{
		Status: &[]domain.ProcessStatus{domain.ProcessStatusRunning}[0],
	}
	return sm.ListProcesses(filter)
}

// GetStatistics returns state manager statistics
func (sm *StateManager) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"total_processes": atomic.LoadInt64(&sm.totalProcesses),
		"total_queries":   atomic.LoadInt64(&sm.totalQueries),
		"total_updates":   atomic.LoadInt64(&sm.totalUpdates),
		"total_events":    atomic.LoadInt64(&sm.totalEvents),
		"active_processes": len(sm.processes),
		"query_queue_size": len(sm.queryCh),
		"update_queue_size": len(sm.updateCh),
		"event_queue_size": len(sm.eventsCh),
	}
}

// run is the main state manager loop (single writer)
func (sm *StateManager) run() {
	defer close(sm.done)
	
	log.Println("StateManager: Started")
	
	// Load existing processes from storage
	sm.loadExistingProcesses()
	
	for {
		select {
		case query := <-sm.queryCh:
			atomic.AddInt64(&sm.totalQueries, 1)
			sm.handleQuery(query)
			
		case update := <-sm.updateCh:
			atomic.AddInt64(&sm.totalUpdates, 1)
			sm.handleUpdate(update)
			
		case event := <-sm.eventsCh:
			atomic.AddInt64(&sm.totalEvents, 1)
			sm.handleEvent(event)
			
		case <-sm.ctx.Done():
			log.Println("StateManager: Stopping...")
			sm.saveAllProcesses()
			log.Println("StateManager: Stopped")
			return
		}
	}
}

// handleQuery processes state queries
func (sm *StateManager) handleQuery(query StateQuery) {
	var response StateQueryResponse
	
	switch query.Type {
	case "get_process":
		processID, ok := query.Filter.(string)
		if !ok {
			response = StateQueryResponse{Success: false, Error: fmt.Errorf("invalid process ID")}
			break
		}
		
		state, exists := sm.processes[processID]
		if !exists {
			response = StateQueryResponse{Success: false, Error: fmt.Errorf("process not found")}
			break
		}
		
		response = StateQueryResponse{Success: true, Data: state.Process}
		
	case "get_actor":
		processID, ok := query.Filter.(string)
		if !ok {
			response = StateQueryResponse{Success: false, Error: fmt.Errorf("invalid process ID")}
			break
		}
		
		state, exists := sm.processes[processID]
		if !exists {
			response = StateQueryResponse{Success: false, Error: fmt.Errorf("process not found")}
			break
		}
		
		response = StateQueryResponse{Success: true, Data: state.Actor}
		
	case "list_processes":
		filter, ok := query.Filter.(domain.ProcessFilter)
		if !ok {
			response = StateQueryResponse{Success: false, Error: fmt.Errorf("invalid filter")}
			break
		}
		
		processes := sm.filterProcesses(filter)
		response = StateQueryResponse{Success: true, Data: processes}
		
	case "count_processes":
		count := len(sm.processes)
		response = StateQueryResponse{Success: true, Data: count}
		
	default:
		response = StateQueryResponse{Success: false, Error: fmt.Errorf("unknown query type: %s", query.Type)}
	}
	
	// Send response (with timeout to prevent blocking)
	select {
	case query.ResponseCh <- response:
	case <-time.After(1 * time.Second):
		log.Printf("StateManager: Query response timeout for type %s", query.Type)
	}
}

// handleUpdate processes state updates
func (sm *StateManager) handleUpdate(update StateUpdate) {
	switch update.Type {
	case "register":
		data, ok := update.Data.(map[string]interface{})
		if !ok {
			log.Printf("StateManager: Invalid register data for process %s", update.ProcessID[:8])
			return
		}
		
		process, ok := data["process"].(*domain.Process)
		if !ok {
			log.Printf("StateManager: Invalid process in register data for %s", update.ProcessID[:8])
			return
		}
		
		actor, ok := data["actor"].(*ProcessActor)
		if !ok {
			log.Printf("StateManager: Invalid actor in register data for %s", update.ProcessID[:8])
			return
		}
		
		sm.processes[update.ProcessID] = &ProcessState{
			Process:   process,
			Actor:     actor,
			CreatedAt: update.Timestamp,
			UpdatedAt: update.Timestamp,
		}
		
		atomic.AddInt64(&sm.totalProcesses, 1)
		log.Printf("StateManager: Registered process %s (%s)", update.ProcessID[:8], process.Name)
		
		// Save to storage asynchronously
		go func() {
			if err := sm.storage.SaveProcess(process.ProjectID, process); err != nil {
				log.Printf("StateManager: Failed to save process %s: %v", update.ProcessID[:8], err)
			}
		}()
		
	case "unregister":
		state, exists := sm.processes[update.ProcessID]
		if exists {
			delete(sm.processes, update.ProcessID)
			atomic.AddInt64(&sm.totalProcesses, -1)
			log.Printf("StateManager: Unregistered process %s (%s)", 
				update.ProcessID[:8], state.Process.Name)
		}
		
	case "update_status":
		state, exists := sm.processes[update.ProcessID]
		if exists {
			if newStatus, ok := update.Data.(domain.ProcessStatus); ok {
				state.Process.Status = newStatus
				state.UpdatedAt = update.Timestamp
				
				// Save to storage asynchronously
				go func() {
					if err := sm.storage.SaveProcess(state.Process.ProjectID, state.Process); err != nil {
						log.Printf("StateManager: Failed to save process status %s: %v", 
							update.ProcessID[:8], err)
					}
				}()
			}
		}
		
	default:
		log.Printf("StateManager: Unknown update type: %s", update.Type)
	}
}

// handleEvent processes events from process actors
func (sm *StateManager) handleEvent(event ProcessEvent) {
	switch event.Type {
	case "process_started":
		sm.updateProcessFromEvent(event.ProcessID, domain.ProcessStatusRunning, event)
		
	case "process_stopped":
		// Determine final status from event data
		var status domain.ProcessStatus = domain.ProcessStatusStopped
		
		if data, ok := event.Data.(map[string]interface{}); ok {
			if eventStatus, ok := data["status"].(domain.ProcessStatus); ok {
				status = eventStatus
			}
		}
		
		sm.updateProcessFromEvent(event.ProcessID, status, event)
		
	case "process_crashed":
		sm.updateProcessFromEvent(event.ProcessID, domain.ProcessStatusCrashed, event)
		
	case "actor_stopped":
		// Actor has stopped, but we keep the process state for queries
		log.Printf("StateManager: Actor stopped for process %s", event.ProcessID[:8])
	}
}

// updateProcessFromEvent updates process status based on an event
func (sm *StateManager) updateProcessFromEvent(processID string, status domain.ProcessStatus, event ProcessEvent) {
	update := StateUpdate{
		Type:      "update_status",
		ProcessID: processID,
		Data:      status,
		Timestamp: event.Timestamp,
	}
	
	// Process update immediately since we're already in the state manager loop
	sm.handleUpdate(update)
}

// filterProcesses applies a filter to return matching processes
func (sm *StateManager) filterProcesses(filter domain.ProcessFilter) []*domain.Process {
	var result []*domain.Process
	
	for _, state := range sm.processes {
		process := state.Process
		
		// Apply filters
		if filter.ProjectID != nil && process.ProjectID != *filter.ProjectID {
			continue
		}
		
		if filter.Status != nil && process.Status != *filter.Status {
			continue
		}
		
		if filter.Type != nil && process.Type != *filter.Type {
			continue
		}
		
		result = append(result, process)
	}
	
	return result
}

// loadExistingProcesses loads processes from storage on startup
func (sm *StateManager) loadExistingProcesses() {
	log.Println("StateManager: Loading existing processes from storage...")
	
	processes, err := sm.storage.ListProcesses(domain.ProcessFilter{})
	if err != nil {
		log.Printf("StateManager: Failed to load processes: %v", err)
		return
	}
	
	loaded := 0
	for _, process := range processes {
		// Only load processes that were running (they're now stopped)
		if process.Status == domain.ProcessStatusRunning {
			process.Status = domain.ProcessStatusStopped
		}
		
		// Create state without actor (actor will be created if process is restarted)
		sm.processes[process.ID] = &ProcessState{
			Process:   process,
			Actor:     nil,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		
		loaded++
	}
	
	atomic.AddInt64(&sm.totalProcesses, int64(loaded))
	log.Printf("StateManager: Loaded %d processes from storage", loaded)
}

// saveAllProcesses saves all processes to storage (used during shutdown)
func (sm *StateManager) saveAllProcesses() {
	log.Printf("StateManager: Saving %d processes to storage...", len(sm.processes))
	
	saved := 0
	for _, state := range sm.processes {
		if err := sm.storage.SaveProcess(state.Process.ProjectID, state.Process); err != nil {
			log.Printf("StateManager: Failed to save process %s: %v", state.Process.ID[:8], err)
		} else {
			saved++
		}
	}
	
	log.Printf("StateManager: Saved %d processes to storage", saved)
}

// Stop gracefully stops the state manager
func (sm *StateManager) Stop() {
	log.Println("StateManager: Stopping...")
	sm.cancel()
	
	select {
	case <-sm.done:
		log.Println("StateManager: Stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("StateManager: Stop timeout")
	}
}

// IsHealthy checks if the state manager is healthy
func (sm *StateManager) IsHealthy() bool {
	select {
	case <-sm.ctx.Done():
		return false
	default:
		// Check if queues are severely backed up
		queryUsage := float64(len(sm.queryCh)) / 100.0
		updateUsage := float64(len(sm.updateCh)) / 100.0
		eventUsage := float64(len(sm.eventsCh)) / 1000.0
		
		return queryUsage < 0.8 && updateUsage < 0.8 && eventUsage < 0.8
	}
}