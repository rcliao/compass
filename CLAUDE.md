# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Compass is an MCP (Model Context Protocol) server designed for AI-assisted planning and context management. It bridges the gap between AI planning and implementation by serving as a persistent context keeper that maintains project understanding across sessions.

**Key Principles:**
- Dumb storage, smart retrieval
- Context sufficiency over context maximalism
- Agent-agnostic design
- Zero external dependencies for core operations

## Project Structure

```
compass-mcp/
├── cmd/
│   └── compass/
│       └── main.go          # MCP server entry point
├── internal/
│   ├── domain/              # Domain models (Task, Project, Discovery, etc.)
│   ├── storage/             # File-based and memory storage implementations
│   ├── service/             # Business logic layers
│   ├── mcp/                 # MCP protocol handlers
│   └── search/              # Hybrid search implementations
├── pkg/
│   └── compass/             # Public API
└── docs/
    └── requirements/
        └── v0.md           # Complete project requirements
```

## Domain Models

The core domain consists of:
- **Task**: Unit of work with context, criteria, and card information
- **Project**: Container for related tasks and planning sessions
- **Process**: Managed subprocess with lifecycle, logs, and health monitoring
- **ProcessGroup**: Collection of related processes for coordinated management
- **Discovery**: Insights learned during development
- **Decision**: Choices made with rationale and alternatives
- **PlanningSession**: Structured planning phases

## Storage System

Uses file-based storage with this structure:
```
.compass/
├── config.json
└── projects/
    └── {project-id}/
        ├── project.json
        ├── tasks.json
        ├── processes.json
        ├── process-groups.json
        ├── discoveries.json
        ├── decisions.json
        ├── planning/
        └── index/
```

## MCP Commands

The server implements these MCP command categories:

### **Project Management**
- `compass.project.create` - Create new projects
- `compass.project.list` - List all projects  
- `compass.project.current` - Get current project
- `compass.project.set_current` - Set current project

### **Task & TODO Management**
- `compass.task.create` - Create tasks with 3 C's structure
- `compass.task.update` - Update task details
- `compass.task.list` - List tasks with filtering
- `compass.todo.create` - Create TODO items
- `compass.todo.complete` - Mark TODOs as completed
- `compass.todo.list` - List TODOs with filters
- `compass.todo.overdue` - Get overdue TODOs

### **Process Management**
- `compass.process.create` - Create process definitions
- `compass.process.start` - Start processes with validation
- `compass.process.stop` - Stop running processes gracefully
- `compass.process.list` - List processes with filtering
- `compass.process.get` - Get detailed process information
- `compass.process.logs` - Retrieve process logs with limits
- `compass.process.status` - Get formatted process status and health
- `compass.process.update` - Update process configuration
- `compass.process.group.create` - Create process groups
- `compass.process.group.start` - Start all processes in group
- `compass.process.group.stop` - Stop all processes in group

### **Context & Intelligence**
- `compass.context.get` - Get full context for tasks
- `compass.context.search` - Search tasks by query
- `compass.context.check` - Check context sufficiency
- `compass.next` - Get next recommended task
- `compass.blockers` - Get all blocked tasks

### **Planning & Discovery**
- `compass.planning.start` - Start planning sessions
- `compass.discovery.add` - Record development insights
- `compass.decision.record` - Record decisions with rationale

## MCP Discovery

Compass implements complete MCP discovery via three endpoints that must be kept synchronized:

### **Tools Discovery (`tools/list`)**
All MCP commands are exposed as discoverable tools with JSON schemas:

```bash
# List all available tools (22 total)
echo '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | ./bin/compass

# Filter by category
echo '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}' | ./bin/compass | \
  jq '.result.tools[] | select(.name | startswith("compass_process"))'
```

**Current Tool Categories:**
- **Project Management** (4 tools): `compass_project_*`
- **Task/TODO Management** (6 tools): `compass_todo_*`, `compass_task_*`
- **Process Management** (11 tools): `compass_process_*`
- **Context & Intelligence** (2 tools): `compass_next`, `compass_blockers`, `compass_context_*`

### **Resources Discovery (`resources/list`)**
Static data accessible via URI-based resources:

```bash
# List all available resources (10 total)
echo '{"jsonrpc": "2.0", "id": 1, "method": "resources/list"}' | ./bin/compass

# Read a specific resource
echo '{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": {"uri": "compass://processes"}}' | ./bin/compass
```

**Current Resources:**
- `compass://projects` - All projects list
- `compass://todos` - All TODO items
- `compass://current` - Current active project
- `compass://overdue` - Overdue TODO items
- `compass://blockers` - Blocked TODO items
- `compass://processes` - All managed processes
- `compass://processes/running` - Running processes only
- `compass://processes/failed` - Failed/crashed processes
- `compass://process-groups` - Process groups
- `compass://processes/logs` - Process logs summary

### **Prompts Discovery (`prompts/list`)**
Guided workflows for common development scenarios:

```bash
# List all available prompts (5 total)
echo '{"jsonrpc": "2.0", "id": 1, "method": "prompts/list"}' | ./bin/compass
```

**Current Prompts:**
- `setup-dev-environment` - Complete development environment setup
- `debug-process-issues` - Debug process startup/runtime issues
- `start-testing-workflow` - Comprehensive testing setup
- `optimize-build-process` - Build process optimization
- `monitor-services` - Service health monitoring

### **Discovery Maintenance**

**⚠️ CRITICAL: When adding new features, always update ALL THREE discovery endpoints:**

1. **Add to `tools/list`** in `internal/mcp/transport.go`:
   ```go
   tools := []map[string]interface{}{
       // Add new tool definition with proper JSON schema
       {
           "name": "compass_new_feature",
           "description": "Description of new feature",
           "inputSchema": map[string]interface{}{
               // Proper JSON schema definition
           },
       },
   }
   ```

2. **Add to `resources/list`** if applicable:
   ```go
   resources := []map[string]interface{}{
       // Add new resource if it provides queryable data
       {
           "uri": "compass://new-resource",
           "name": "New Resource",
           "description": "Description of resource",
           "mimeType": "text/markdown", // or application/json
       },
   }
   ```

3. **Add resource handler** in `handleResourceRead`:
   ```go
   case "compass://new-resource":
       result, err = t.server.HandleCommand("compass.new.command", params)
   ```

4. **Add to `prompts/list`** for common workflows:
   ```go
   prompts := []map[string]interface{}{
       // Add new prompt for guided workflows
       {
           "name": "new-workflow-prompt",
           "description": "Description of workflow",
           "arguments": []map[string]interface{}{
               // Argument definitions
           },
       },
   }
   ```

### **Testing Discovery**

Always test all three endpoints after changes:

```bash
# Test tools discovery
echo '{"jsonrpc": "2.0", "method": "tools/list"}' | ./bin/compass | jq '.result.tools | length'

# Test resources discovery  
echo '{"jsonrpc": "2.0", "method": "resources/list"}' | ./bin/compass | jq '.result.resources | length'

# Test prompts discovery
echo '{"jsonrpc": "2.0", "method": "prompts/list"}' | ./bin/compass | jq '.result.prompts | length'

# Test specific resource reading
echo '{"jsonrpc": "2.0", "method": "resources/read", "params": {"uri": "compass://processes"}}' | ./bin/compass
```

All discovery endpoints have proper JSON schemas and validation for IDE support.

## Development Checklist

When adding new features to Compass, follow this checklist to ensure proper MCP integration:

### **✅ New Command Implementation Checklist**

1. **[ ] Domain Model** - Add/update domain models in `internal/domain/`
2. **[ ] Storage Layer** - Implement storage interface methods in `internal/storage/`
3. **[ ] Service Layer** - Add business logic in `internal/service/`
4. **[ ] MCP Handler** - Add command handler in `internal/mcp/server.go`
5. **[ ] MCP Tool Definition** - Add to `tools` array in `handleToolsList()`
6. **[ ] MCP Tool Mapping** - Add case to switch statement in `handleToolCall()`
7. **[ ] Resource Exposure** - Add to `resources` array in `handleResourcesList()` (if data resource)
8. **[ ] Resource Handler** - Add case to switch in `handleResourceRead()` (if data resource)
9. **[ ] Prompt Definition** - Add to `prompts` array in `handlePromptsList()` (if workflow)
10. **[ ] Documentation Update** - Update this CLAUDE.md file with new commands
11. **[ ] Testing** - Test all discovery endpoints and functionality

### **🧪 Pre-Commit Testing**

Always run these tests before committing new MCP features:

```bash
# 1. Build binary
go build -o bin/compass cmd/compass/main.go

# 2. Test discovery endpoints
echo '{"jsonrpc": "2.0", "method": "tools/list"}' | ./bin/compass | jq '.result.tools | length'
echo '{"jsonrpc": "2.0", "method": "resources/list"}' | ./bin/compass | jq '.result.resources | length'  
echo '{"jsonrpc": "2.0", "method": "prompts/list"}' | ./bin/compass | jq '.result.prompts | length'

# 3. Test new command functionality
echo 'compass.your.new.command {"test": "data"}' | ./bin/compass --cli

# 4. Test resource reading (if applicable)
echo '{"jsonrpc": "2.0", "method": "resources/read", "params": {"uri": "compass://new-resource"}}' | ./bin/compass
```

## Development Commands

```bash
# Initialize Go module
go mod init github.com/yourusername/compass-mcp

# Create project structure
mkdir -p cmd/compass internal/{domain,storage,service,mcp,search} pkg/compass

# Run tests
go test ./...

# Build binary
go build -o bin/compass cmd/compass/main.go

# Run MCP server
./bin/compass

# Test MCP discovery (should show current counts)
echo '{"jsonrpc": "2.0", "method": "tools/list"}' | ./bin/compass | jq '.result.tools | length'  # Should be 22+
echo '{"jsonrpc": "2.0", "method": "resources/list"}' | ./bin/compass | jq '.result.resources | length'  # Should be 10+
echo '{"jsonrpc": "2.0", "method": "prompts/list"}' | ./bin/compass | jq '.result.prompts | length'  # Should be 5+
```

## Git Workflow

**CRITICAL: Always create a git commit after each meaningful chunk of changes**

When working on compass development, follow this mandatory workflow:

1. **Make changes** - Implement features, fix bugs, or make improvements
2. **Test changes** - Build and verify functionality works correctly
3. **Commit immediately** - Never leave changes uncommitted for extended periods

### Commit Guidelines

- **Commit frequency**: After every logical unit of work (feature, fix, refactor)
- **Commit scope**: Each commit should represent a complete, working change
- **Commit messages**: Follow the existing pattern with clear, descriptive messages
- **Testing**: Always build and test before committing

### Example Workflow

```bash
# 1. Make changes to code
# 2. Build and test
go build -o bin/compass cmd/compass/main.go

# 3. Stage and commit changes
git add [changed-files]
git commit -m "descriptive commit message

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"

# 4. Continue with next chunk of work
```

**Why this matters**:
- Prevents loss of work
- Maintains clear development history  
- Enables easy rollback if issues arise
- Facilitates code review and collaboration
- Ensures incremental progress is preserved

## Dependencies

Minimal external dependencies:
- `github.com/google/uuid` - ID generation
- `github.com/tidwall/gjson` - JSON handling
- `github.com/stretchr/testify` - Testing framework
- MCP Go library (when available)

## Architecture Notes

- **Concurrency**: Uses `sync.RWMutex` for thread-safe operations
- **Caching**: In-memory caching for frequently accessed tasks
- **Search**: Hybrid search combining keyword, header, and structural matching
- **Context Headers**: Generated contextual summaries for tasks
- **Atomic Operations**: File operations use atomic writes with temporary files
- **Process Management**: Real-time log capture, health monitoring, graceful shutdown
- **Working Directory**: Inherits from agent launch directory, not MCP binary location

## Development Phases

1. **Core Foundation**: ✅ MCP server scaffold, file storage, task CRUD
2. **Context System**: ✅ Header generation, search, retrieval methods
3. **Process Management**: ✅ Subprocess lifecycle, log capture, health monitoring
4. **Planning Integration**: ✅ Session management, discovery tracking
5. **Production Readiness**: Error handling, logging, performance optimization

## Process Management Examples

### **Web Development Workflow**
```bash
# Create and start development server
compass.process.create {
  "name": "Next.js Dev Server",
  "command": "npm",
  "args": ["run", "dev"],
  "type": "web-server",
  "port": 3000
}

# Start the process
compass.process.start {"id": "server-id"}

# Monitor startup logs
compass.process.logs {"id": "server-id", "limit": 20}

# Check server health
compass.process.status {"id": "server-id"}
```

### **Multi-Service Development**
```bash
# Create process group for full stack
compass.process.group.create {
  "name": "Full Stack Dev",
  "description": "Frontend and backend services"
}

# Start all services at once
compass.process.group.start {"id": "group-id"}

# Monitor all processes
compass.process.list {"status": "running"}
```

### **Build Process Monitoring**
```bash
# Create build watcher
compass.process.create {
  "name": "Webpack Watcher",
  "command": "npx",
  "args": ["webpack", "--watch"],
  "type": "build-tool"
}

# Check for compilation errors in logs
compass.process.logs {"id": "build-id", "limit": 50}
```

## MCP Capabilities Inventory

*Last Updated: 2025-07-22*

### **Tools Summary (22 total)**

| Category | Count | Commands |
|----------|--------|----------|
| **Project Management** | 4 | `compass_project_create`, `compass_project_list`, `compass_project_current`, `compass_project_set_current` |
| **Task/TODO Management** | 6 | `compass_todo_create`, `compass_todo_list`, `compass_todo_complete`, `compass_todo_overdue`, `compass_task_*` |
| **Process Management** | 11 | `compass_process_create`, `compass_process_start`, `compass_process_stop`, `compass_process_list`, `compass_process_get`, `compass_process_logs`, `compass_process_status`, `compass_process_update`, `compass_process_group_*` |
| **Context & Intelligence** | 1 | `compass_context_search`, `compass_next`, `compass_blockers` |

### **Resources Summary (10 total)**

| URI | Description | MIME Type | Status |
|-----|-------------|-----------|---------|
| `compass://projects` | All projects list | `application/json` | ✅ Working |
| `compass://todos` | All TODO items | `application/json` | ✅ Working |
| `compass://current` | Current active project | `application/json` | ✅ Working |
| `compass://overdue` | Overdue TODO items | `application/json` | ✅ Working |
| `compass://blockers` | Blocked TODO items | `application/json` | ✅ Working |
| `compass://processes` | All managed processes | `text/markdown` | ✅ Working |
| `compass://processes/running` | Running processes only | `text/markdown` | ✅ Working |
| `compass://processes/failed` | Failed/crashed processes | `text/markdown` | ✅ Working |
| `compass://process-groups` | Process groups | `application/json` | ⚠️ Placeholder |
| `compass://processes/logs` | Process logs summary | `text/markdown` | ⚠️ Placeholder |

### **Prompts Summary (5 total)**

| Prompt Name | Purpose | Arguments | Status |
|-------------|---------|-----------|---------|
| `setup-dev-environment` | Complete development environment setup | `project_type`, `framework`, `include_database` | ✅ Defined |
| `debug-process-issues` | Debug process startup/runtime issues | `process_name`, `issue_type` | ✅ Defined |
| `start-testing-workflow` | Comprehensive testing setup | `test_type`, `watch_mode` | ✅ Defined |
| `optimize-build-process` | Build process optimization | `build_tool` | ✅ Defined |
| `monitor-services` | Service health monitoring | `include_logs`, `alert_threshold` | ✅ Defined |

### **Discovery Endpoint Status**

```bash
# Current discovery counts (as of last update)
tools/list:     22 tools     (✅ All working)
resources/list: 10 resources (8 working, 2 placeholders)
prompts/list:   5 prompts    (✅ All defined, implementation pending)
```

### **Update Instructions**

When this inventory becomes outdated:

1. **Regenerate counts**:
   ```bash
   echo '{"jsonrpc": "2.0", "method": "tools/list"}' | ./bin/compass | jq '.result.tools | length'
   echo '{"jsonrpc": "2.0", "method": "resources/list"}' | ./bin/compass | jq '.result.resources | length'
   echo '{"jsonrpc": "2.0", "method": "prompts/list"}' | ./bin/compass | jq '.result.prompts | length'
   ```

2. **Update the "Last Updated" date** at the top of this section

3. **Add new capabilities** to the appropriate summary table

4. **Update status indicators**: ✅ Working, ⚠️ Placeholder, ❌ Broken

## Testing Strategy

- Unit tests for all domain models and services
- Integration tests for MCP command handlers
- Performance benchmarks for search and storage operations
- Memory storage implementation for fast testing
- Process management integration tests with real subprocesses