# Compass MCP Server

A Context-Oriented Memory system for Planning and Software Systems, implemented as an MCP (Model Context Protocol) server in Go.

## Overview

Compass is designed to bridge the gap between AI-assisted planning and implementation by serving as a persistent context keeper that helps AI coding agents maintain project understanding across sessions.

### Key Concepts

- **Dumb Storage, Smart Retrieval**: Focus on intelligent ways to find and connect information rather than complex storage
- **Context Sufficiency**: Ensure tasks have all necessary information before work begins
- **Agent-Agnostic**: Works with any AI coding assistant that supports MCP
- **Zero Dependencies**: Core operations require no external services or databases

### Why Compass?

Traditional task management tools lack the context awareness needed for AI-assisted development. Compass solves this by:

1. **Maintaining Context**: Tasks include file references, dependencies, and acceptance criteria
2. **Tracking Discoveries**: Insights found during development automatically update related tasks
3. **Recording Decisions**: Technical choices are documented with rationale and impact
4. **Intelligent Recommendations**: Suggests next tasks based on dependencies and project state
5. **Planning Sessions**: Structured approach to project planning with full traceability

## Features

### Core Features
- **File-based Storage**: Persistent storage using `.compass/` directory structure
- **MCP Protocol**: Standard Model Context Protocol for AI agent communication  
- **Task Management**: Create, update, and track development tasks with context
- **Project Organization**: Group tasks into projects with goals and descriptions
- **Thread-safe Operations**: Concurrent access with proper synchronization
- **Atomic File Operations**: Data integrity through atomic writes

### Process Management
- **Development Server Management**: Start, stop, and monitor web servers, APIs, and build tools
- **Process Templates**: 15+ predefined templates for common development scenarios
- **Port Conflict Detection**: Automatic detection with intelligent alternative suggestions
- **Environment Variable Support**: Secure validation and management of environment variables
- **Process Groups**: Coordinate multiple related services together
- **Log Capture**: Real-time log capture with searchable history

### Enhanced Developer Experience
- **Quick Todo Creation**: Simplified `compass.todo.quick` for fast task creation
- **Connection Recovery**: Robust error handling and automatic reconnection
- **Enhanced Error Messages**: Descriptive errors with troubleshooting hints
- **Command Validation**: Executable checks and working directory validation
- **Template System**: Pre-configured setups for React, Node.js, Python, databases, and more

## Quick Start

### Build and Run

```bash
# Build the server
go build -o bin/compass cmd/compass/main.go

# Run the interactive CLI
./bin/compass
```

### MCP Integration

#### Claude Code

Add to your Claude Code configuration file (`~/.config/claude-code/mcp_servers.json`):

```json
{
  "mcpServers": {
    "compass": {
      "command": "/path/to/your/compass/bin/compass",
      "args": [],
      "env": {}
    }
  }
}
```

#### Cursor

Add to your Cursor MCP settings (`~/.cursor/mcp_servers.json`):

```json
{
  "mcpServers": {
    "compass": {
      "command": "/path/to/your/compass/bin/compass",
      "args": [],
      "env": {}
    }
  }
}
```

#### Usage in AI Assistants

Once configured, you can use Compass commands directly in your AI assistant:

```
# In Claude Code or Cursor chat
Can you create a new project for my web app?
> Uses: compass.project.create

What's the next task I should work on?
> Uses: compass.next

Search for authentication-related tasks
> Uses: compass.context.search
```

### Basic Usage

```bash
# Create a project
compass.project.create {"name":"My Project","description":"A test project","goal":"Learn Compass"}

# Set as current project (many commands use current project by default)
compass.project.set_current {"id":"<project-id>"}

# Create a task
compass.task.create {"projectId":"<project-id>","title":"Setup","description":"Initial setup"}

# List tasks
compass.task.list {"projectId":"<project-id>"}
```

### Context System Usage

```bash
# Search for tasks using hybrid search
compass.context.search {"query":"authentication","limit":5}

# Get full context for a task including dependencies and related tasks
compass.context.get {"taskId":"<task-id>"}

# Check if a task has sufficient context to begin work
compass.context.check {"taskId":"<task-id>"}
# Returns: {"taskId":"...","sufficient":true/false,"missing":[...],"stale":[...]}

# Get next recommended task based on dependencies and priority
compass.next {}
# Returns the best task to work on next

# Get all blocked tasks
compass.blockers {}
```

### Planning Session Usage

```bash
# Start a planning session
compass.planning.start {"name":"Sprint Planning Week 1"}

# List all planning sessions
compass.planning.list {}

# Complete a planning session
compass.planning.complete {"id":"<session-id>"}
```

### Discovery and Decision Tracking

```bash
# Record a discovery during development
compass.discovery.add {
  "insight":"Users prefer OAuth login over custom authentication",
  "impact":"high",
  "source":"research",
  "affectedTaskIds":["<task-id-1>","<task-id-2>"]
}

# List all discoveries
compass.discovery.list {}

# Record a technical decision
compass.decision.record {
  "question":"Which database should we use?",
  "choice":"PostgreSQL",
  "rationale":"Better JSON support and proven scalability",
  "alternatives":["MySQL","MongoDB"],
  "reversible":true,
  "affectedTaskIds":["<task-id>"]
}

# List all decisions
compass.decision.list {}
```

### Process Management

```bash
# Create a process using a template
compass.process.create {
  "name": "React Dev Server",
  "template": "react-dev",
  "workingDir": "/path/to/project"
}

# Or create a custom process
compass.process.create {
  "name": "Custom Server",
  "command": "npm",
  "args": ["run", "dev"],
  "type": "web-server",
  "port": 3000,
  "environment": {
    "NODE_ENV": "development"
  }
}

# Start the process
compass.process.start {"id": "<process-id>"}

# Monitor logs
compass.process.logs {"id": "<process-id>", "limit": 50}

# Check status
compass.process.status {"id": "<process-id>"}

# Stop when done
compass.process.stop {"id": "<process-id>"}
```

### Available Process Templates

**Frontend Development:**
- `react-dev` - React development server (port 3000)
- `next-dev` - Next.js development server (port 3000)
- `vite-dev` - Vite development server (port 5173)
- `webpack-dev` - Webpack development server (port 8080)

**Backend Development:**
- `node-server` - Node.js server (port 8000)
- `express-dev` - Express development server (port 3001)
- `go-server` - Go server (port 8080)

**Python Development:**
- `python-server` - Python HTTP server (port 8000)
- `flask-dev` - Flask development server (port 5000)
- `django-dev` - Django development server (port 8000)

**Databases:**
- `postgres` - PostgreSQL server (port 5432)
- `redis` - Redis server (port 6379)
- `mysql` - MySQL server (port 3306)

**Tools:**
- `tailwind-watch` - Tailwind CSS watcher
- `jest-watch` - Jest test runner in watch mode

### Quick Todo Creation

```bash
# Simple todo creation without full 3 C's structure
compass.todo.quick {
  "title": "Fix login bug",
  "description": "Users can't log in with email",
  "priority": "high",
  "labels": ["bug", "frontend"]
}
```

### Project Intelligence

```bash
# Generate comprehensive project summary with analytics
compass.project.summary {}
# Returns:
# - Task statistics by status and confidence
# - Recent discoveries and decisions
# - Velocity trends (improving/stable/declining)
# - Context health score (excellent/good/fair/poor)
# - Intelligent recommendations for next actions
```

### Advanced Task Management

```bash
# Create a task with full context
compass.task.create {
  "projectId":"<project-id>",
  "title":"Implement user authentication",
  "description":"Add JWT-based authentication to the API",
  "files":["auth.go","middleware.go"],
  "dependencies":["<setup-task-id>"],
  "acceptance":[
    "Users can register with email/password",
    "JWT tokens are properly validated",
    "Protected endpoints require authentication"
  ]
}

# Update task status
compass.task.update {
  "id":"<task-id>",
  "updates":{
    "status":"in-progress",
    "confidence":"high"
  }
}

# Mark task as blocked
compass.task.update {
  "id":"<task-id>",
  "updates":{
    "status":"blocked",
    "blockers":["Waiting for API design approval"]
  }
}
```

## Architecture

```
internal/
├── domain/         # Core business models (Task, Project, Discovery, Decision)
├── storage/        # Storage implementations (memory, file)
├── service/        # Business logic layer
├── mcp/           # MCP protocol handlers
└── search/        # Search implementations (future)
```

## Storage Structure

```
.compass/
├── config.json
└── projects/
    └── {project-id}/
        ├── project.json
        ├── tasks.json
        ├── discoveries.json
        ├── decisions.json
        ├── planning/
        └── index/
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

### Project Structure

- `cmd/compass/` - Main entry point and CLI
- `internal/domain/` - Domain models and interfaces
- `internal/storage/` - Storage implementations
- `internal/service/` - Business logic services  
- `internal/mcp/` - MCP server and handlers
- `pkg/compass/` - Public API (future)

## MCP Commands

### Project Commands
- `compass.project.create` - Create a new project
- `compass.project.list` - List all projects
- `compass.project.current` - Get current project
- `compass.project.set_current` - Set current project

### Task Commands  
- `compass.task.create` - Create a new task
- `compass.task.update` - Update a task
- `compass.task.list` - List tasks with filtering
- `compass.task.get` - Get a specific task
- `compass.task.delete` - Delete a task

### Context Commands
- `compass.context.get` - Get full task context with dependencies and related tasks
- `compass.context.search` - Hybrid search across title, description, files, and dependencies
- `compass.context.check` - Check if task has sufficient context information

### Intelligent Queries
- `compass.next` - Get next recommended task based on dependencies and priority
- `compass.blockers` - Get all blocked tasks in current project

### Process Commands
- `compass.process.create` - Create a new process (supports templates)
- `compass.process.start` - Start a process
- `compass.process.stop` - Stop a process  
- `compass.process.list` - List processes with filtering
- `compass.process.get` - Get detailed process information
- `compass.process.logs` - Retrieve process logs
- `compass.process.status` - Get formatted process status and health
- `compass.process.update` - Update process configuration
- `compass.process.group.create` - Create a process group
- `compass.process.group.start` - Start all processes in a group
- `compass.process.group.stop` - Stop all processes in a group

### TODO Commands
- `compass.todo.create` - Create a TODO with full 3 C's structure
- `compass.todo.quick` - Create a simple TODO (new!)
- `compass.todo.complete` - Mark TODO as completed
- `compass.todo.list` - List TODOs with filtering
- `compass.todo.overdue` - Get overdue TODOs

### Planning Commands
- `compass.planning.start` - Start a new planning session
- `compass.planning.list` - List all planning sessions
- `compass.planning.get` - Get planning session details
- `compass.planning.complete` - Complete a planning session
- `compass.planning.abort` - Abort a planning session
- `compass.discovery.add` - Record a new discovery
- `compass.discovery.list` - List all discoveries
- `compass.decision.record` - Record a decision
- `compass.decision.list` - List all decisions
- `compass.project.summary` - Generate project summary with analytics

## Example Workflow

Here's a typical workflow showing how all features work together:

```bash
# 1. Create and setup a project
compass.project.create {"name":"E-commerce API","description":"RESTful API for online store","goal":"Build scalable e-commerce backend"}
compass.project.set_current {"id":"<project-id>"}

# 2. Start a planning session
compass.planning.start {"name":"Initial Architecture Planning"}

# 3. Create initial tasks during planning
compass.task.create {
  "title":"Design API architecture",
  "description":"Define RESTful endpoints and data models",
  "acceptance":["OpenAPI spec created","Data models documented"]
}

compass.task.create {
  "title":"Setup database",
  "description":"Configure PostgreSQL with migrations",
  "files":["database.go","migrations/"],
  "acceptance":["Database connected","Initial migrations created"]
}

# 4. Record a discovery during research
compass.discovery.add {
  "insight":"Stripe API requires webhook endpoints for payment confirmations",
  "impact":"high",
  "source":"research",
  "affectedTaskIds":["<api-design-task-id>"]
}

# 5. Make and record a technical decision
compass.decision.record {
  "question":"Which payment processor to use?",
  "choice":"Stripe",
  "rationale":"Best documentation and developer experience",
  "alternatives":["PayPal","Square"],
  "reversible":true,
  "affectedTaskIds":["<api-design-task-id>"]
}

# 6. Search for related tasks
compass.context.search {"query":"payment","limit":5}

# 7. Get next recommended task
compass.next {}

# 8. Check task context before starting work
compass.context.check {"taskId":"<task-id>"}

# 9. View project progress and insights
compass.project.summary {}
# Returns comprehensive analytics including:
# - Task completion velocity
# - Context health score
# - Recent decisions and discoveries
# - Actionable recommendations

# 10. Complete the planning session
compass.planning.complete {"id":"<session-id>"}
```

## Phase 4 Complete: Production Readiness & Process Management

### ✅ Completed Features

**Phase 1: Foundation**
- ✅ Complete Go module setup
- ✅ Core domain models
- ✅ Memory and file storage
- ✅ Basic MCP server
- ✅ CLI interface
- ✅ Unit and integration tests

**Phase 2: Context System**
- ✅ Contextual header generation for tasks
- ✅ Hybrid search (keyword, header, structural)
- ✅ Context retrieval with task relationships
- ✅ Staleness detection and verification
- ✅ Context sufficiency checking
- ✅ Intelligent task recommendations
- ✅ Next task suggestions with scoring

**Phase 3: Planning Integration**
- ✅ Planning session management with structured phases
- ✅ Discovery tracking and insights recording
- ✅ Decision recording with rationale and alternatives
- ✅ Intelligent project summaries and analytics
- ✅ Session-based task organization
- ✅ Automated context updates from discoveries/decisions

**Phase 4: Production Readiness & Process Management**
- ✅ Comprehensive error handling with panic recovery
- ✅ Connection recovery and health monitoring
- ✅ Process management with 15+ development templates
- ✅ Port conflict detection and resolution
- ✅ Environment variable validation and security
- ✅ Enhanced parameter validation with descriptive errors
- ✅ Quick todo creation for improved developer experience
- ✅ Command executable validation and path checking
- ✅ Timeout handling and connection state management

### System Status: Production Ready ✅
The Compass MCP server is now production-ready with robust error handling, comprehensive process management, and enhanced developer experience features.

## License

MIT License - see LICENSE file for details.