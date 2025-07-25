# Compass Process Management Guide

This guide explains how to use Compass MCP's process management capabilities to run and monitor long-running processes like web servers, build tools, and other development services.

## Overview

The process management system allows coding agents to:
- Start and stop processes with proper lifecycle management
- Use predefined templates for common development scenarios
- Capture and retrieve process logs with real-time monitoring
- Monitor process health and status with automatic recovery
- Manage groups of related processes
- Handle process crashes with automatic restart policies
- Detect and resolve port conflicts automatically
- Validate commands and environment variables securely

## Core Concepts

### Process Types
- `web-server`: HTTP/HTTPS servers
- `api-server`: API services
- `build-tool`: Build processes (webpack, vite, etc.)
- `watcher`: File watchers
- `test`: Test runners
- `database`: Database servers
- `custom`: Any other process type

### Process States
- `pending`: Process created but not started
- `starting`: Process is starting up
- `running`: Process is actively running
- `stopping`: Process is shutting down
- `stopped`: Process stopped normally
- `failed`: Process exited with error
- `crashed`: Process terminated unexpectedly

## MCP Commands

### Creating a Process with Templates

Using predefined templates (recommended):

```json
compass.process.create {
  "name": "React Dev Server",
  "template": "react-dev",
  "workingDir": "/path/to/project"
}
```

Available templates:
- **Frontend**: `react-dev`, `next-dev`, `vite-dev`, `webpack-dev`
- **Backend**: `node-server`, `express-dev`, `go-server`
- **Python**: `python-server`, `flask-dev`, `django-dev`
- **Databases**: `postgres`, `redis`, `mysql`
- **Tools**: `tailwind-watch`, `jest-watch`

### Creating a Custom Process

```json
compass.process.create {
  "name": "Dev Server",
  "command": "npm",
  "args": ["run", "dev"],
  "type": "web-server",
  "port": 3000,
  "workingDir": "/path/to/project",  // Optional: defaults to agent's working directory
  "environment": {
    "NODE_ENV": "development"
  }
}
```

**Enhanced Features:**
- **Command Validation**: Automatically checks if commands are executable
- **Port Conflict Detection**: Suggests alternative ports if specified port is in use
- **Environment Variable Validation**: Validates variable names and warns about sensitive data
- **Working Directory Validation**: Ensures specified directories exist

**Working Directory Behavior:**
- If `workingDir` is specified: Uses that directory
- If `workingDir` is omitted: Uses directory where the coding agent was launched
- This ensures `npm run dev` runs in your project directory, not the MCP binary location

### Starting a Process

```json
compass.process.start {
  "id": "process-id-here"
}
```

### Getting Process Status

```json
compass.process.status {
  "id": "process-id-here"
}
```

Returns formatted status including:
- Current state and uptime
- PID and port information
- Health check status
- Restart policy details

### Viewing Process Logs

```json
compass.process.logs {
  "id": "process-id-here",
  "limit": 100
}
```

Returns the last N log entries with timestamps and type indicators:
- `[OUT]`: Standard output
- `[ERR]`: Standard error
- `[SYS]`: System messages

### Listing All Processes

```json
compass.process.list {
  "projectId": "current-project-id",
  "status": "running"
}
```

### Stopping a Process

```json
compass.process.stop {
  "id": "process-id-here"
}
```

Attempts graceful shutdown with SIGTERM, falls back to SIGKILL if needed.

## Process Groups

### Creating a Process Group

```json
compass.process.group.create {
  "name": "Full Stack Dev",
  "description": "Frontend and backend servers",
  "processIds": ["frontend-id", "backend-id"]
}
```

### Starting/Stopping Groups

```json
compass.process.group.start {
  "id": "group-id"
}

compass.process.group.stop {
  "id": "group-id"
}
```

## Example Workflows

### 1. Running a Web Development Server

```bash
# Create the process
compass.process.create {
  "name": "Next.js Dev",
  "command": "npm",
  "args": ["run", "dev"],
  "type": "web-server",
  "port": 3000,
  "environment": {
    "NODE_ENV": "development"
  }
}

# Start it
compass.process.start {"id": "abc123"}

# Check status
compass.process.status {"id": "abc123"}

# View logs
compass.process.logs {"id": "abc123", "limit": 50}

# Stop when done
compass.process.stop {"id": "abc123"}
```

### 2. Running Multiple Services

```bash
# Create frontend process
compass.process.create {
  "name": "Frontend",
  "command": "npm",
  "args": ["run", "dev"],
  "workingDir": "./frontend",
  "type": "web-server",
  "port": 3000
}

# Create backend process
compass.process.create {
  "name": "Backend API",
  "command": "python",
  "args": ["app.py"],
  "workingDir": "./backend",
  "type": "api-server",
  "port": 8000,
  "environment": {
    "FLASK_ENV": "development"
  }
}

# Create a group
compass.process.group.create {
  "name": "Full Stack",
  "description": "Frontend and Backend services",
  "processIds": ["frontend-id", "backend-id"]
}

# Start all at once
compass.process.group.start {"id": "group-id"}
```

### 3. Running Tests with Log Monitoring

```bash
# Create test process
compass.process.create {
  "name": "Test Suite",
  "command": "npm",
  "args": ["test", "--watch"],
  "type": "test"
}

# Start and monitor
compass.process.start {"id": "test-id"}

# Check logs periodically
compass.process.logs {"id": "test-id", "limit": 20}
```

## Best Practices

1. **Always specify working directory**: Ensures processes run in the correct context
2. **Use appropriate process types**: Helps with categorization and filtering
3. **Set port numbers for servers**: Makes it easier to track which process uses which port
4. **Monitor logs regularly**: Check for errors or issues during development
5. **Use process groups**: Manage related services together
6. **Clean up stopped processes**: List and review processes periodically

## Integration with Coding Agents

Coding agents can use these commands to:

1. **Start development servers** before testing changes
2. **Monitor build processes** for compilation errors
3. **Run test suites** and capture results
4. **Manage database servers** for integration testing
5. **Coordinate multiple services** for full-stack development

Example agent workflow:
```python
# 1. Create and start web server
process = create_process("web-server", "npm run dev")
start_process(process.id)

# 2. Wait for server to be ready
while get_status(process.id) != "running":
    time.sleep(1)

# 3. Run tests against the server
test_results = run_tests()

# 4. Check server logs for errors
logs = get_logs(process.id, limit=100)
check_for_errors(logs)

# 5. Stop server when done
stop_process(process.id)
```

## Troubleshooting

### Process Won't Start

**Command Not Found:**
- Error: `command not found or not executable`
- Solution: Ensure the command exists in PATH or use absolute path
- The system automatically validates executable availability

**Working Directory Issues:**
- Error: `working directory does not exist`
- Solution: Verify the path exists and is accessible
- Use relative paths from your project directory

**Port Conflicts:**
- Error: `port 3000 is already in use. Suggested alternatives: [3001, 3002, 3003]`
- Solution: Use one of the suggested ports or stop the conflicting process
- The system automatically detects conflicts and suggests alternatives

**Environment Variable Issues:**
- Error: `invalid character in environment variable name`
- Solution: Use only letters, numbers, and underscores in variable names
- Variable names cannot start with digits

### Process Crashes Immediately
- Review logs for error messages: `compass.process.logs {"id": "process-id"}`
- Check if dependencies are installed (npm install, pip install, etc.)
- Verify template compatibility with your project structure

### Connection Issues
The system now includes robust error recovery:
- Automatic panic recovery prevents server crashes
- Connection timeouts are handled gracefully
- Broken pipe errors are detected and managed
- Transport layer includes connection health monitoring

### Can't Stop Process
- Process may have already terminated
- Check process status first: `compass.process.status {"id": "process-id"}`
- System attempts graceful shutdown (SIGTERM) before force kill (SIGKILL)
- Improved cleanup ensures proper resource deallocation

## Future Enhancements

Planned improvements include:
- Real-time log streaming
- Advanced health check configurations
- Process resource monitoring (CPU, memory)
- Automatic port assignment
- Process templates for common setups