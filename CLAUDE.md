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
        ├── discoveries.json
        ├── decisions.json
        ├── planning/
        └── index/
```

## MCP Commands

The server implements these MCP command categories:
- **Project**: `compass.project.create`, `compass.project.list`, `compass.project.current`
- **Task**: `compass.task.create`, `compass.task.update`, `compass.task.list`
- **Context**: `compass.context.get`, `compass.context.search`, `compass.context.check`
- **Planning**: `compass.planning.start`, `compass.discovery.add`, `compass.decision.record`
- **Intelligent**: `compass.next`, `compass.blockers`

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
```

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

## Development Phases

1. **Core Foundation**: ✅ MCP server scaffold, file storage, task CRUD
2. **Context System**: ✅ Header generation, search, retrieval methods
3. **Planning Integration**: Session management, discovery tracking
4. **Production Readiness**: Error handling, logging, performance

## Testing Strategy

- Unit tests for all domain models and services
- Integration tests for MCP command handlers
- Performance benchmarks for search and storage operations
- Memory storage implementation for fast testing