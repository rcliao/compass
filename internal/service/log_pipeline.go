package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/rcliao/compass/internal/domain"
)

// =============================================================================
// LOG PIPELINE - Async log processing with batching (NO BLOCKING I/O!)
// =============================================================================

// LogBuffer manages in-memory logs for a single process
type LogBuffer struct {
	processID string
	logs      []*domain.ProcessLog
	maxSize   int
	// No mutex needed - single writer (LogPipeline)
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(processID string, maxSize int) *LogBuffer {
	return &LogBuffer{
		processID: processID,
		logs:      make([]*domain.ProcessLog, 0, maxSize),
		maxSize:   maxSize,
	}
}

// Add adds a log entry to the buffer
func (lb *LogBuffer) Add(entry *domain.ProcessLog) {
	lb.logs = append(lb.logs, entry)
	
	// Rotate if buffer is full
	if len(lb.logs) > lb.maxSize {
		// Keep the last 75% of logs
		keepCount := (lb.maxSize * 3) / 4
		copy(lb.logs, lb.logs[len(lb.logs)-keepCount:])
		lb.logs = lb.logs[:keepCount]
	}
}

// GetLogs returns a copy of logs (last N entries)
func (lb *LogBuffer) GetLogs(limit int) []*domain.ProcessLog {
	if limit <= 0 || limit > len(lb.logs) {
		limit = len(lb.logs)
	}
	
	if limit == 0 {
		return []*domain.ProcessLog{}
	}
	
	start := len(lb.logs) - limit
	if start < 0 {
		start = 0
	}
	
	// Return a copy to prevent data races
	result := make([]*domain.ProcessLog, len(lb.logs[start:]))
	copy(result, lb.logs[start:])
	
	return result
}

// Count returns the number of logs in the buffer
func (lb *LogBuffer) Count() int {
	return len(lb.logs)
}

// LogPipeline processes logs asynchronously with batching
type LogPipeline struct {
	// Input channel for log entries
	inputCh chan LogEntry
	
	// Storage interface
	storage ProcessStorage
	
	// In-memory buffers for fast reads (no mutex - single writer)
	buffers map[string]*LogBuffer
	
	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	
	// Statistics
	totalLogsProcessed int64
	totalLogsDropped   int64
	totalBatchesSaved  int64
	
	// Configuration
	batchSize       int
	flushInterval   time.Duration
	maxBufferSize   int
	channelCapacity int
}

// LogPipelineConfig holds configuration for the log pipeline
type LogPipelineConfig struct {
	BatchSize       int           // Max logs per batch
	FlushInterval   time.Duration // Max time before flushing batch
	MaxBufferSize   int           // Max logs per process buffer
	ChannelCapacity int           // Input channel buffer size
}

// DefaultLogPipelineConfig returns default configuration
func DefaultLogPipelineConfig() LogPipelineConfig {
	return LogPipelineConfig{
		BatchSize:       50,
		FlushInterval:   100 * time.Millisecond,
		MaxBufferSize:   10000,
		ChannelCapacity: 1000,
	}
}

// NewLogPipeline creates a new log pipeline
func NewLogPipeline(storage ProcessStorage, config LogPipelineConfig) *LogPipeline {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &LogPipeline{
		inputCh:         make(chan LogEntry, config.ChannelCapacity),
		storage:         storage,
		buffers:         make(map[string]*LogBuffer),
		ctx:             ctx,
		cancel:          cancel,
		done:            make(chan struct{}),
		batchSize:       config.BatchSize,
		flushInterval:   config.FlushInterval,
		maxBufferSize:   config.MaxBufferSize,
		channelCapacity: config.ChannelCapacity,
	}
}

// Start begins the log pipeline processing
func (lp *LogPipeline) Start() {
	go lp.run()
}

// SendLog sends a log entry to the pipeline (non-blocking)
func (lp *LogPipeline) SendLog(entry LogEntry) bool {
	select {
	case lp.inputCh <- entry:
		return true
	default:
		// Pipeline is full - drop the log to prevent blocking actors
		lp.totalLogsDropped++
		log.Printf("LogPipeline: Dropped log for process %s (channel full)", entry.ProcessID[:8])
		return false
	}
}

// GetLogs retrieves logs for a process from in-memory buffer
func (lp *LogPipeline) GetLogs(processID string, limit int) ([]*domain.ProcessLog, error) {
	buffer, exists := lp.buffers[processID]
	if !exists {
		// Try to load from storage if no buffer exists
		return lp.storage.GetProcessLogs(processID, limit)
	}
	
	// Get from in-memory buffer
	logs := buffer.GetLogs(limit)
	
	// If we don't have enough logs in memory, supplement from storage
	if len(logs) < limit {
		storageLogs, err := lp.storage.GetProcessLogs(processID, limit-len(logs))
		if err == nil {
			// Merge storage logs (older) with buffer logs (newer)
			allLogs := make([]*domain.ProcessLog, 0, len(storageLogs)+len(logs))
			allLogs = append(allLogs, storageLogs...)
			allLogs = append(allLogs, logs...)
			
			// Return last N logs
			if len(allLogs) > limit {
				start := len(allLogs) - limit
				allLogs = allLogs[start:]
			}
			
			return allLogs, nil
		}
	}
	
	return logs, nil
}

// GetStatistics returns pipeline statistics
func (lp *LogPipeline) GetStatistics() map[string]interface{} {
	stats := map[string]interface{}{
		"total_logs_processed": lp.totalLogsProcessed,
		"total_logs_dropped":   lp.totalLogsDropped,
		"total_batches_saved":  lp.totalBatchesSaved,
		"active_buffers":       len(lp.buffers),
		"channel_capacity":     lp.channelCapacity,
		"channel_length":       len(lp.inputCh),
	}
	
	// Add per-process buffer sizes
	bufferSizes := make(map[string]int)
	for processID, buffer := range lp.buffers {
		bufferSizes[processID] = buffer.Count()
	}
	stats["buffer_sizes"] = bufferSizes
	
	return stats
}

// run is the main pipeline processing loop
func (lp *LogPipeline) run() {
	defer close(lp.done)
	defer lp.flushAllBuffers()
	
	log.Println("LogPipeline: Started")
	
	// Batch for storage
	var batch []LogEntry
	flushTimer := time.NewTicker(lp.flushInterval)
	defer flushTimer.Stop()
	
	for {
		select {
		case entry := <-lp.inputCh:
			// Add to in-memory buffer
			lp.addToBuffer(entry)
			
			// Add to storage batch
			batch = append(batch, entry)
			lp.totalLogsProcessed++
			
			// Flush batch if it's full
			if len(batch) >= lp.batchSize {
				lp.flushBatch(batch)
				batch = nil
			}
			
		case <-flushTimer.C:
			// Periodic flush of pending batch
			if len(batch) > 0 {
				lp.flushBatch(batch)
				batch = nil
			}
			
		case <-lp.ctx.Done():
			// Final flush on shutdown
			if len(batch) > 0 {
				lp.flushBatch(batch)
			}
			log.Println("LogPipeline: Stopped")
			return
		}
	}
}

// addToBuffer adds a log entry to the appropriate in-memory buffer
func (lp *LogPipeline) addToBuffer(entry LogEntry) {
	buffer, exists := lp.buffers[entry.ProcessID]
	if !exists {
		buffer = NewLogBuffer(entry.ProcessID, lp.maxBufferSize)
		lp.buffers[entry.ProcessID] = buffer
	}
	
	// Convert to domain object
	processLog := &domain.ProcessLog{
		ProcessID: entry.ProcessID,
		Type:      entry.Type,
		Message:   entry.Message,
		Timestamp: entry.Timestamp,
	}
	
	buffer.Add(processLog)
}

// flushBatch saves a batch of logs to storage (asynchronously)
func (lp *LogPipeline) flushBatch(batch []LogEntry) {
	if len(batch) == 0 {
		return
	}
	
	// Convert to domain objects
	processLogs := make([]*domain.ProcessLog, len(batch))
	for i, entry := range batch {
		processLogs[i] = &domain.ProcessLog{
			ProcessID: entry.ProcessID,
			Type:      entry.Type,
			Message:   entry.Message,
			Timestamp: entry.Timestamp,
		}
	}
	
	// Save asynchronously to prevent blocking the pipeline
	go func() {
		start := time.Now()
		if err := lp.storage.SaveProcessLogs(processLogs); err != nil {
			log.Printf("LogPipeline: Failed to save batch of %d logs: %v", len(processLogs), err)
		} else {
			lp.totalBatchesSaved++
			duration := time.Since(start)
			
			// Log slow saves
			if duration > 500*time.Millisecond {
				log.Printf("LogPipeline: Slow batch save: %d logs in %v", len(processLogs), duration)
			}
		}
	}()
}

// flushAllBuffers saves all buffered logs to storage (used during shutdown)
func (lp *LogPipeline) flushAllBuffers() {
	log.Printf("LogPipeline: Flushing %d buffers on shutdown", len(lp.buffers))
	
	var wg sync.WaitGroup
	for processID, buffer := range lp.buffers {
		if buffer.Count() == 0 {
			continue
		}
		
		wg.Add(1)
		go func(pid string, buf *LogBuffer) {
			defer wg.Done()
			
			// Save remaining logs
			logs := buf.GetLogs(0) // Get all logs
			if len(logs) > 0 {
				if err := lp.storage.SaveProcessLogs(logs); err != nil {
					log.Printf("LogPipeline: Failed to flush logs for process %s: %v", pid[:8], err)
				} else {
					log.Printf("LogPipeline: Flushed %d logs for process %s", len(logs), pid[:8])
				}
			}
		}(processID, buffer)
	}
	
	// Wait for all flushes to complete (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("LogPipeline: All buffers flushed successfully")
	case <-time.After(10 * time.Second):
		log.Println("LogPipeline: Flush timeout - some logs may be lost")
	}
}

// CleanupBuffer removes the buffer for a process (when process is deleted)
func (lp *LogPipeline) CleanupBuffer(processID string) {
	delete(lp.buffers, processID)
}

// Stop gracefully stops the log pipeline
func (lp *LogPipeline) Stop() {
	log.Println("LogPipeline: Stopping...")
	lp.cancel()
	
	// Wait for pipeline to finish with timeout
	select {
	case <-lp.done:
		log.Println("LogPipeline: Stopped gracefully")
	case <-time.After(15 * time.Second):
		log.Println("LogPipeline: Stop timeout")
	}
}

// Health check for the log pipeline
func (lp *LogPipeline) IsHealthy() bool {
	select {
	case <-lp.ctx.Done():
		return false
	default:
		// Check if channel is severely backed up
		channelUsage := float64(len(lp.inputCh)) / float64(lp.channelCapacity)
		return channelUsage < 0.9 // Healthy if less than 90% full
	}
}