# Compass MCP Server

A Context-Oriented Memory system for Planning and Software Systems, implemented as an MCP (Model Context Protocol) server in Go.

## Overview

Compass is designed to bridge the gap between AI-assisted planning and implementation by serving as a persistent context keeper that helps AI coding agents maintain project understanding across sessions.

## Features

- **File-based Storage**: Persistent storage using `.compass/` directory structure
- **MCP Protocol**: Standard Model Context Protocol for AI agent communication  
- **Task Management**: Create, update, and track development tasks with context
- **Project Organization**: Group tasks into projects with goals and descriptions
- **Thread-safe Operations**: Concurrent access with proper synchronization
- **Atomic File Operations**: Data integrity through atomic writes

## Quick Start

### Build and Run

```bash
# Build the server
go build -o bin/compass cmd/compass/main.go

# Run the interactive CLI
./bin/compass
```

### Basic Usage

```bash
# Create a project
compass.project.create {"name":"My Project","description":"A test project","goal":"Learn Compass"}

# Create a task
compass.task.create {"projectId":"<project-id>","title":"Setup","description":"Initial setup"}

# List tasks
compass.task.list {"projectId":"<project-id>"}
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

## Next Steps

This foundation provides:
- ✅ Complete Go module setup
- ✅ Core domain models
- ✅ Memory and file storage
- ✅ Basic MCP server
- ✅ CLI interface
- ✅ Unit and integration tests

Ready for Phase 2 implementation:
- Context header generation
- Hybrid search functionality
- Planning session management
- Discovery and decision tracking

## License

MIT License - see LICENSE file for details.