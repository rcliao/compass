package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCPTransport handles JSON-RPC 2.0 communication over stdio
type MCPTransport struct {
	reader       *bufio.Reader
	writer       io.Writer
	server       *MCPServer
	lastActivity time.Time
	connected    bool
	mu           sync.Mutex
}

// NewMCPTransport creates a new MCP transport over stdio
func NewMCPTransport(server *MCPServer) *MCPTransport {
	return &MCPTransport{
		reader:       bufio.NewReader(os.Stdin),
		writer:       os.Stdout,
		server:       server,
		lastActivity: time.Now(),
		connected:    true,
	}
}

// Start begins the MCP server transport loop
func (t *MCPTransport) Start() error {
	for {
		// Wrap each request processing in panic recovery
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("MCP transport: Panic recovered: %v", r)
					// Try to send error response to client
					errorResp := &JSONRPCResponse{
						JSONRPC: "2.0",
						Error: &JSONRPCError{
							Code:    InternalError,
							Message: "Internal server error",
						},
					}
					t.sendResponse(errorResp)
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()
			
			// Read line from stdin (with timeout)
			lineChan := make(chan []byte, 1)
			errChan := make(chan error, 1)
			
			go func() {
				line, err := t.reader.ReadBytes('\n')
				if err != nil {
					errChan <- err
				} else {
					lineChan <- line
				}
			}()
			
			// Wait for input or timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			var line []byte
			select {
			case line = <-lineChan:
				t.updateActivity()
			case err := <-errChan:
				if err == io.EOF {
					log.Println("MCP transport: client disconnected")
					return io.EOF // Normal termination
				}
				return fmt.Errorf("failed to read from stdin: %w", err)
			case <-ctx.Done():
				// Check if we should timeout or continue waiting
				if t.shouldTimeout() {
					log.Println("MCP transport: connection timeout")
					return fmt.Errorf("connection timeout")
				}
				// Continue waiting for long-running processes
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
				defer cancel()
				select {
				case line = <-lineChan:
					t.updateActivity()
				case err := <-errChan:
					if err == io.EOF {
						return io.EOF
					}
					return fmt.Errorf("failed to read from stdin: %w", err)
				case <-ctx.Done():
					return fmt.Errorf("connection timeout after extended wait")
				}
			}

			// Process the JSON-RPC request
			response := t.processRequest(line)
			
			// Send response if it's not a notification
			if response != nil {
				if err := t.sendResponse(response); err != nil {
					// Check for broken pipe
					if strings.Contains(err.Error(), "broken pipe") || 
					   strings.Contains(err.Error(), "connection reset") {
						log.Printf("MCP transport: Client disconnected: %v", err)
						return io.EOF
					}
					return fmt.Errorf("failed to send response: %w", err)
				}
			}
			
			return nil
		}()
		
		// Handle errors from the request processing
		if err != nil {
			if err == io.EOF {
				return nil // Clean disconnect
			}
			// Log error but continue processing
			log.Printf("MCP transport: Error processing request: %v", err)
		}
	}
}

// processRequest processes a JSON-RPC request and returns a response
func (t *MCPTransport) processRequest(data []byte) *JSONRPCResponse {
	// Parse JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    ParseError,
				Message: "Parse error",
				Data:    err.Error(),
			},
		}
	}

	// Check JSON-RPC version
	if req.JSONRPC != "2.0" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    InvalidRequest,
				Message: "Invalid Request - JSON-RPC 2.0 required",
			},
		}
	}

	// Handle initialization and standard MCP methods
	switch req.Method {
	case "initialize":
		return t.handleInitialize(req)
	case "initialized":
		// Notification - no response needed
		return nil
	case "shutdown":
		return t.handleShutdown(req)
	case "exit":
		// Notification - exit the server with cleanup
		fmt.Fprintf(os.Stderr, "Transport: Received exit command, calling server shutdown...\n")
		t.server.Shutdown()
		fmt.Fprintf(os.Stderr, "Transport: Server shutdown completed, exiting...\n")
		os.Exit(0)
		return nil
	default:
		// Handle Compass-specific methods
		return t.handleCompassMethod(req)
	}
}

// handleInitialize handles the MCP initialize request
func (t *MCPTransport) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	// Parse initialization parameters
	type InitParams struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Roots struct {
				ListChanged bool `json:"listChanged"`
			} `json:"roots,omitempty"`
		} `json:"capabilities,omitempty"`
		ClientInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo,omitempty"`
	}

	var params InitParams
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    InvalidParams,
					Message: "Invalid params",
					Data:    err.Error(),
				},
			}
		}
	}

	// Return server capabilities
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
			"resources": map[string]interface{}{
				"subscribe":   false,
				"listChanged": false,
			},
			"prompts": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "compass",
			"version": "1.0.0",
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// updateActivity updates the last activity timestamp
func (t *MCPTransport) updateActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastActivity = time.Now()
}

// shouldTimeout checks if the connection should timeout based on inactivity
func (t *MCPTransport) shouldTimeout() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Don't timeout if we've had recent activity
	return time.Since(t.lastActivity) > 10*time.Minute
}

// isConnected checks if the transport is still connected
func (t *MCPTransport) isConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connected
}

// setConnected sets the connection status
func (t *MCPTransport) setConnected(connected bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.connected = connected
}

// handleShutdown handles the MCP shutdown request
func (t *MCPTransport) handleShutdown(req JSONRPCRequest) *JSONRPCResponse {
	t.setConnected(false)
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  nil,
	}
}

// handleCompassMethod handles Compass-specific method calls
func (t *MCPTransport) handleCompassMethod(req JSONRPCRequest) *JSONRPCResponse {
	// Handle tool requests
	if req.Method == "tools/list" {
		return t.handleToolsList(req)
	}
	if req.Method == "tools/call" {
		return t.handleToolCall(req)
	}

	// Handle resource requests
	if req.Method == "resources/list" {
		return t.handleResourcesList(req)
	}
	if req.Method == "resources/read" {
		return t.handleResourceRead(req)
	}

	// Handle prompt requests
	if req.Method == "prompts/list" {
		return t.handlePromptsList(req)
	}
	if req.Method == "prompts/get" {
		return t.handlePromptGet(req)
	}

	// Direct method calls (legacy compatibility)
	result, err := t.server.HandleCommand(req.Method, req.Params)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    MethodNotFound,
				Message: err.Error(),
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleToolsList handles MCP tools list requests
func (t *MCPTransport) handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	// Define all available Compass tools
	tools := []map[string]interface{}{
		// Project commands
		{
			"name":        "compass_project_create",
			"description": "Create a new project",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":        map[string]interface{}{"type": "string", "description": "Project name"},
					"description": map[string]interface{}{"type": "string", "description": "Project description"},
					"goal":        map[string]interface{}{"type": "string", "description": "Project goal"},
				},
				"required": []string{"name", "description", "goal"},
			},
		},
		{
			"name":        "compass_project_list",
			"description": "List all projects",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false},
		},
		{
			"name":        "compass_project_current",
			"description": "Get current project",
			"inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false},
		},
		{
			"name":        "compass_project_set_current",
			"description": "Set current project",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Project ID"},
				},
				"required": []string{"id"},
			},
		},
		// Task/TODO commands
		{
			"name":        "compass_todo_create",
			"description": "Create a new TODO item with Card, Context, and Criteria (3 C's)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId": map[string]interface{}{"type": "string", "description": "Project ID (optional if current project is set)"},
					"card": map[string]interface{}{
						"type": "object",
						"description": "The task card (what needs to be done)",
						"required": []string{"title", "description"},
						"properties": map[string]interface{}{
							"title":          map[string]interface{}{"type": "string", "description": "Clear, actionable task title"},
							"description":    map[string]interface{}{"type": "string", "description": "Detailed description of what needs to be done"},
							"priority":       map[string]interface{}{"type": "string", "enum": []string{"low", "medium", "high"}, "description": "Priority level"},
							"dueDate":        map[string]interface{}{"type": "string", "format": "date-time", "description": "Due date"},
							"estimatedHours": map[string]interface{}{"type": "number", "description": "Estimated hours to complete"},
							"labels":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Labels/tags for categorization"},
							"assignedTo":     map[string]interface{}{"type": "string", "description": "Person assigned to this task"},
						},
					},
					"context": map[string]interface{}{
						"type": "object",
						"description": "The task context (where and with what)",
						"required": []string{"dependencies", "assumptions"},
						"properties": map[string]interface{}{
							"files": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{"type": "string"},
								"description": "Files involved in this task",
							},
							"dependencies": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{"type": "string"},
								"description": "Task IDs or external dependencies",
							},
							"assumptions": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{"type": "string"},
								"description": "Assumptions being made",
							},
						},
					},
					"criteria": map[string]interface{}{
						"type": "object",
						"description": "The task criteria (how to know it's done)",
						"required": []string{"acceptance"},
						"properties": map[string]interface{}{
							"acceptance": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{"type": "string"},
								"description": "At least 2 acceptance criteria defining done",
							},
							"verification": map[string]interface{}{
								"type": "array",
								"items": map[string]interface{}{"type": "string"},
								"description": "How to verify the criteria are met",
							},
						},
					},
				},
				"required": []string{"card", "context", "criteria"},
			},
		},
		{
			"name":        "compass_todo_list",
			"description": "List TODO items with filters",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId":  map[string]interface{}{"type": "string", "description": "Filter by project ID"},
					"status":     map[string]interface{}{"type": "string", "enum": []string{"planned", "in_progress", "completed", "blocked"}, "description": "Filter by status"},
					"priority":   map[string]interface{}{"type": "string", "enum": []string{"low", "medium", "high"}, "description": "Filter by priority"},
					"labels":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Filter by labels"},
					"assignedTo": map[string]interface{}{"type": "string", "description": "Filter by assignee"},
					"limit":      map[string]interface{}{"type": "integer", "description": "Limit results"},
				},
				"additionalProperties": false,
			},
		},
		{
			"name":        "compass_todo_complete",
			"description": "Mark TODO as completed with required verification evidence",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "TODO ID"},
					"evidence": map[string]interface{}{
						"type":        "array",
						"minItems":    1,
						"description": "Verification evidence (minimum 1 per acceptance criteria)",
						"items": map[string]interface{}{
							"type": "object",
							"required": []string{"evidence"},
							"properties": map[string]interface{}{
								"evidence": map[string]interface{}{
									"type":        "string",
									"minLength":   10,
									"description": "What was tested/verified (memo notes)",
								},
								"testType": map[string]interface{}{
									"type":        "string",
									"description": "Type of test performed (e.g., manual, automated, integration)",
								},
								"testResults": map[string]interface{}{
									"type":        "string",
									"description": "Results or output from the test",
								},
								"relatedCriteria": map[string]interface{}{
									"type":        "array",
									"items":       map[string]interface{}{"type": "integer"},
									"description": "Acceptance criteria indices this evidence addresses",
								},
							},
						},
					},
					"completionNotes": map[string]interface{}{
						"type":        "string",
						"description": "Overall completion summary and notes",
					},
				},
				"required": []string{"id", "evidence"},
			},
		},
		{
			"name":        "compass_todo_overdue",
			"description": "Get overdue TODO items",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId": map[string]interface{}{"type": "string", "description": "Filter by project ID"},
				},
				"additionalProperties": false,
			},
		},
		// Context commands
		{
			"name":        "compass_context_search",
			"description": "Search tasks by query",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":     map[string]interface{}{"type": "string", "description": "Search query"},
					"projectId": map[string]interface{}{"type": "string", "description": "Filter by project ID"},
					"limit":     map[string]interface{}{"type": "integer", "description": "Limit results"},
					"offset":    map[string]interface{}{"type": "integer", "description": "Offset for pagination"},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "compass_next",
			"description": "Get next recommended task",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId": map[string]interface{}{"type": "string", "description": "Project ID (optional if current project is set)"},
					"exclude":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Task IDs to exclude"},
				},
				"additionalProperties": false,
			},
		},
		{
			"name":        "compass_blockers",
			"description": "Get all blocked tasks",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId": map[string]interface{}{"type": "string", "description": "Project ID (optional if current project is set)"},
				},
				"additionalProperties": false,
			},
		},
		// Process commands
		{
			"name":        "compass_process_create",
			"description": "Create a new process",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId":   map[string]interface{}{"type": "string", "description": "Project ID (optional if current project is set)"},
					"name":        map[string]interface{}{"type": "string", "description": "Process name"},
					"command":     map[string]interface{}{"type": "string", "description": "Command to execute"},
					"args":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Command arguments"},
					"workingDir":  map[string]interface{}{"type": "string", "description": "Working directory"},
					"environment": map[string]interface{}{"type": "object", "description": "Environment variables"},
					"type":        map[string]interface{}{"type": "string", "enum": []string{"web-server", "api-server", "build-tool", "watcher", "test", "database", "custom"}, "description": "Process type"},
					"port":        map[string]interface{}{"type": "integer", "description": "Port number (for servers)"},
				},
				"required": []string{"name", "command"},
			},
		},
		{
			"name":        "compass_process_start",
			"description": "Start a process",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Process ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "compass_process_stop",
			"description": "Stop a running process",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Process ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "compass_process_list",
			"description": "List processes with optional filters",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId": map[string]interface{}{"type": "string", "description": "Filter by project ID"},
					"status":    map[string]interface{}{"type": "string", "enum": []string{"pending", "starting", "running", "stopping", "stopped", "failed", "crashed"}, "description": "Filter by status"},
					"type":      map[string]interface{}{"type": "string", "enum": []string{"web-server", "api-server", "build-tool", "watcher", "test", "database", "custom"}, "description": "Filter by type"},
				},
				"additionalProperties": false,
			},
		},
		{
			"name":        "compass_process_get",
			"description": "Get process details",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Process ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "compass_process_logs",
			"description": "Get process logs",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":    map[string]interface{}{"type": "string", "description": "Process ID"},
					"limit": map[string]interface{}{"type": "integer", "description": "Number of log entries to return (default: 100)"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "compass_process_status",
			"description": "Get process status and health information",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Process ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "compass_process_update",
			"description": "Update process configuration",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":      map[string]interface{}{"type": "string", "description": "Process ID"},
					"updates": map[string]interface{}{"type": "object", "description": "Fields to update"},
				},
				"required": []string{"id", "updates"},
			},
		},
		{
			"name":        "compass_process_group_create",
			"description": "Create a process group",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"projectId":   map[string]interface{}{"type": "string", "description": "Project ID (optional if current project is set)"},
					"name":        map[string]interface{}{"type": "string", "description": "Group name"},
					"description": map[string]interface{}{"type": "string", "description": "Group description"},
					"processIds":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Process IDs to include"},
				},
				"required": []string{"name", "description"},
			},
		},
		{
			"name":        "compass_process_group_start",
			"description": "Start all processes in a group",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Group ID"},
				},
				"required": []string{"id"},
			},
		},
		{
			"name":        "compass_process_group_stop",
			"description": "Stop all processes in a group",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string", "description": "Group ID"},
				},
				"required": []string{"id"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// handleToolCall handles MCP tool calls
func (t *MCPTransport) handleToolCall(req JSONRPCRequest) *JSONRPCResponse {
	type ToolCallParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	}

	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    InvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	// Convert arguments to JSON for HandleCommand
	var argsJSON json.RawMessage
	if params.Arguments != nil {
		var err error
		argsJSON, err = json.Marshal(params.Arguments)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    InternalError,
					Message: "Failed to serialize arguments",
					Data:    err.Error(),
				},
			}
		}
	}

	// Map MCP tool names back to internal command names
	var commandName string
	switch params.Name {
	case "compass_project_create":
		commandName = "compass.project.create"
	case "compass_project_list":
		commandName = "compass.project.list"
	case "compass_project_current":
		commandName = "compass.project.current"
	case "compass_project_set_current":
		commandName = "compass.project.set_current"
	case "compass_todo_create":
		commandName = "compass.todo.create"
	case "compass_todo_list":
		commandName = "compass.todo.list"
	case "compass_todo_complete":
		commandName = "compass.todo.complete"
	case "compass_todo_overdue":
		commandName = "compass.todo.overdue"
	case "compass_context_search":
		commandName = "compass.context.search"
	case "compass_next":
		commandName = "compass.next"
	case "compass_blockers":
		commandName = "compass.blockers"
	// Process commands
	case "compass_process_create":
		commandName = "compass.process.create"
	case "compass_process_start":
		commandName = "compass.process.start"
	case "compass_process_stop":
		commandName = "compass.process.stop"
	case "compass_process_list":
		commandName = "compass.process.list"
	case "compass_process_get":
		commandName = "compass.process.get"
	case "compass_process_logs":
		commandName = "compass.process.logs"
	case "compass_process_status":
		commandName = "compass.process.status"
	case "compass_process_update":
		commandName = "compass.process.update"
	case "compass_process_group_create":
		commandName = "compass.process.group.create"
	case "compass_process_group_start":
		commandName = "compass.process.group.start"
	case "compass_process_group_stop":
		commandName = "compass.process.group.stop"
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("Unknown tool: %s", params.Name),
			},
		}
	}

	// Call the Compass method
	result, err := t.server.HandleCommand(commandName, argsJSON)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    InternalError,
				Message: err.Error(),
			},
		}
	}

	// Return result in MCP tool call format
	var textContent string
	
	// Check if result is already a string (markdown formatted)
	if str, ok := result.(string); ok {
		textContent = str
	} else {
		// Otherwise serialize to JSON
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    InternalError,
					Message: "Failed to serialize result",
					Data:    err.Error(),
				},
			}
		}
		textContent = string(resultJSON)
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": textContent,
				},
			},
		},
	}
}

// handleResourcesList handles MCP resources list requests
func (t *MCPTransport) handleResourcesList(req JSONRPCRequest) *JSONRPCResponse {
	// Define available Compass resources
	resources := []map[string]interface{}{
		{
			"uri":         "compass://projects",
			"name":        "All Projects",
			"description": "List of all projects in Compass",
			"mimeType":    "application/json",
		},
		{
			"uri":         "compass://todos",
			"name":        "All TODOs",
			"description": "List of all TODO items across all projects",
			"mimeType":    "application/json",
		},
		{
			"uri":         "compass://current",
			"name":        "Current Project",
			"description": "Current active project and its details",
			"mimeType":    "application/json",
		},
		{
			"uri":         "compass://overdue",
			"name":        "Overdue Items",
			"description": "All overdue TODO items",
			"mimeType":    "application/json",
		},
		{
			"uri":         "compass://blockers",
			"name":        "Blocked Items",
			"description": "All blocked TODO items",
			"mimeType":    "application/json",
		},
		// Process Management Resources
		{
			"uri":         "compass://processes",
			"name":        "All Processes",
			"description": "List of all managed processes across all projects",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         "compass://processes/running",
			"name":        "Running Processes",
			"description": "Currently running processes with status information",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         "compass://processes/failed",
			"name":        "Failed Processes",
			"description": "Processes that have failed or crashed",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         "compass://process-groups",
			"name":        "Process Groups",
			"description": "All process groups and their member processes",
			"mimeType":    "application/json",
		},
		{
			"uri":         "compass://processes/logs",
			"name":        "Process Logs Summary",
			"description": "Recent logs from all active processes",
			"mimeType":    "text/markdown",
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"resources": resources,
		},
	}
}

// handleResourceRead handles MCP resource read requests
func (t *MCPTransport) handleResourceRead(req JSONRPCRequest) *JSONRPCResponse {
	type ResourceParams struct {
		URI string `json:"uri"`
	}

	var params ResourceParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    InvalidParams,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	// Handle compass:// URI resources
	var result interface{}
	var err error
	
	switch params.URI {
	case "compass://projects":
		// List all projects
		result, err = t.server.HandleCommand("compass.project.list", nil)
	case "compass://todos":
		// List all TODOs
		result, err = t.server.HandleCommand("compass.todo.list", nil)
	case "compass://current":
		// Get current project
		result, err = t.server.HandleCommand("compass.project.current", nil)
	case "compass://overdue":
		// Get overdue TODOs
		result, err = t.server.HandleCommand("compass.todo.overdue", nil)
	case "compass://blockers":
		// Get blocked items
		result, err = t.server.HandleCommand("compass.blockers", nil)
	// Process Management Resources
	case "compass://processes":
		// List all processes
		result, err = t.server.HandleCommand("compass.process.list", nil)
	case "compass://processes/running":
		// List running processes
		runningFilter := map[string]interface{}{"status": "running"}
		params, _ := json.Marshal(runningFilter)
		result, err = t.server.HandleCommand("compass.process.list", params)
	case "compass://processes/failed":
		// List failed/crashed processes
		failedFilter := map[string]interface{}{"status": "failed"}
		params1, _ := json.Marshal(failedFilter)
		result1, err1 := t.server.HandleCommand("compass.process.list", params1)
		crashedFilter := map[string]interface{}{"status": "crashed"}
		params2, _ := json.Marshal(crashedFilter)
		result2, err2 := t.server.HandleCommand("compass.process.list", params2)
		
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		} else {
			// Combine failed and crashed processes
			result = fmt.Sprintf("# Failed and Crashed Processes\n\n## Failed Processes\n\n%s\n\n## Crashed Processes\n\n%s", result1, result2)
		}
	case "compass://process-groups":
		// List all process groups - need to implement this command
		result = "Process groups listing not yet implemented"
		err = nil
	case "compass://processes/logs":
		// Get recent logs from all processes - would need a special handler
		result = "# Process Logs Summary\n\nCombined process logs not yet implemented.\nUse `compass.process.logs` with specific process ID to get logs for individual processes."
		err = nil
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    InvalidParams,
				Message: fmt.Sprintf("Unknown resource URI: %s", params.URI),
			},
		}
	}
	
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    InternalError,
				Message: fmt.Sprintf("Failed to fetch resource: %s", err.Error()),
			},
		}
	}
	
	// Convert result to text content
	var textContent string
	if str, ok := result.(string); ok {
		textContent = str
	} else {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    InternalError,
					Message: "Failed to serialize resource content",
					Data:    err.Error(),
				},
			}
		}
		textContent = string(resultJSON)
	}
	
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"uri":      params.URI,
					"mimeType": "text/markdown",
					"text":     textContent,
				},
			},
		},
	}
}

// handlePromptsList handles MCP prompts list requests
func (t *MCPTransport) handlePromptsList(req JSONRPCRequest) *JSONRPCResponse {
	// Define available Compass prompts
	prompts := []map[string]interface{}{
		{
			"name":        "setup-dev-environment",
			"description": "Set up a complete development environment with web server, database, and build tools",
			"arguments": []map[string]interface{}{
				{
					"name":        "project_type",
					"description": "Type of project (web, api, fullstack, mobile)",
					"required":    true,
				},
				{
					"name":        "framework",
					"description": "Framework to use (react, vue, express, nextjs, etc.)",
					"required":    false,
				},
				{
					"name":        "include_database",
					"description": "Whether to include database setup",
					"required":    false,
				},
			},
		},
		{
			"name":        "debug-process-issues",
			"description": "Debug common process issues like failed starts, crashes, or performance problems",
			"arguments": []map[string]interface{}{
				{
					"name":        "process_name",
					"description": "Name or ID of the process to debug",
					"required":    true,
				},
				{
					"name":        "issue_type",
					"description": "Type of issue (startup_failure, crash, performance, logs)",
					"required":    false,
				},
			},
		},
		{
			"name":        "start-testing-workflow",
			"description": "Start a complete testing workflow with test runners, coverage, and CI processes",
			"arguments": []map[string]interface{}{
				{
					"name":        "test_type",
					"description": "Type of testing (unit, integration, e2e, all)",
					"required":    true,
				},
				{
					"name":        "watch_mode",
					"description": "Whether to run tests in watch mode",
					"required":    false,
				},
			},
		},
		{
			"name":        "optimize-build-process",
			"description": "Analyze and optimize build processes for better performance",
			"arguments": []map[string]interface{}{
				{
					"name":        "build_tool",
					"description": "Build tool being used (webpack, vite, rollup, etc.)",
					"required":    false,
				},
			},
		},
		{
			"name":        "monitor-services",
			"description": "Monitor all running services and provide health status report",
			"arguments": []map[string]interface{}{
				{
					"name":        "include_logs",
					"description": "Whether to include recent logs in the report",
					"required":    false,
				},
				{
					"name":        "alert_threshold",
					"description": "Alert threshold for service health (low, medium, high)",
					"required":    false,
				},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"prompts": prompts,
		},
	}
}

// handlePromptGet handles MCP prompt requests
func (t *MCPTransport) handlePromptGet(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Error: &JSONRPCError{
			Code:    MethodNotFound,
			Message: "Prompts not yet implemented",
		},
	}
}

// sendResponse sends a JSON-RPC response to stdout
func (t *MCPTransport) sendResponse(response *JSONRPCResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Write to stdout with newline
	if _, err := t.writer.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

// sendNotification sends a JSON-RPC notification
func (t *MCPTransport) sendNotification(method string, params interface{}) error {
	notification := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
	}

	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal notification params: %w", err)
		}
		notification.Params = paramsJSON
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Write to stdout with newline
	if _, err := t.writer.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}