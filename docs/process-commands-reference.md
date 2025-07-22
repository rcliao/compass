# Compass Process Management - Quick Reference

## Process Lifecycle Commands

### Create Process
```json
compass.process.create {
  "name": "Dev Server",
  "command": "npm",
  "args": ["run", "dev"],
  "type": "web-server",           // Optional: web-server, api-server, build-tool, watcher, test, database, custom
  "port": 3000,                   // Optional: port for servers
  "workingDir": "/path/to/project", // Optional: defaults to agent working directory
  "environment": {                // Optional: environment variables
    "NODE_ENV": "development"
  }
}
```

### Start Process
```json
compass.process.start {
  "id": "process-id"
}
```

### Stop Process
```json
compass.process.stop {
  "id": "process-id"
}
```

## Process Information Commands

### List Processes
```json
compass.process.list {
  "projectId": "project-id",      // Optional: filter by project
  "status": "running",            // Optional: pending, starting, running, stopping, stopped, failed, crashed
  "type": "web-server"            // Optional: filter by type
}
```

### Get Process Details
```json
compass.process.get {
  "id": "process-id"
}
```

### Get Process Status (Formatted)
```json
compass.process.status {
  "id": "process-id"
}
```
Returns formatted markdown with status, uptime, health info, etc.

### Get Process Logs
```json
compass.process.logs {
  "id": "process-id",
  "limit": 100                    // Optional: number of log entries (default: 100)
}
```
Returns formatted logs with timestamps and type indicators.

## Process Group Commands

### Create Process Group
```json
compass.process.group.create {
  "name": "Full Stack Dev",
  "description": "Frontend and backend services",
  "processIds": ["frontend-id", "backend-id"]  // Optional: can add processes later
}
```

### Start Process Group
```json
compass.process.group.start {
  "id": "group-id"
}
```

### Stop Process Group
```json
compass.process.group.stop {
  "id": "group-id"
}
```

## Process Configuration

### Update Process
```json
compass.process.update {
  "id": "process-id",
  "updates": {
    "name": "New Name",
    "environment": {"NEW_VAR": "value"},
    "restartPolicy": {
      "enabled": true,
      "maxRetries": 5,
      "retryDelay": 10000000000
    }
  }
}
```

## Process Types

- **web-server**: HTTP/HTTPS servers (Express, Next.js, etc.)
- **api-server**: API services (REST, GraphQL)
- **build-tool**: Build processes (webpack, vite, rollup)
- **watcher**: File watchers (nodemon, chokidar)
- **test**: Test runners (jest, mocha, pytest)
- **database**: Database servers (postgres, redis, mongo)
- **custom**: Any other process type

## Process States

- **pending**: Created but not started
- **starting**: In startup phase
- **running**: Active and healthy
- **stopping**: Shutting down gracefully
- **stopped**: Terminated normally
- **failed**: Exited with error
- **crashed**: Terminated unexpectedly

## Log Types

- **[OUT]**: Standard output
- **[ERR]**: Standard error
- **[SYS]**: System messages from Compass

## Common Usage Patterns

### Development Server Setup
```bash
# 1. Create development server
compass.process.create {
  "name": "Dev Server",
  "command": "npm", 
  "args": ["run", "dev"],
  "type": "web-server"
}

# 2. Start and monitor
compass.process.start {"id": "server-id"}
compass.process.logs {"id": "server-id", "limit": 20}

# 3. Check status
compass.process.status {"id": "server-id"}
```

### Multi-Service Development
```bash
# 1. Create multiple processes
compass.process.create {"name": "Frontend", "command": "npm", "args": ["run", "dev"], "workingDir": "./client"}
compass.process.create {"name": "Backend", "command": "npm", "args": ["start"], "workingDir": "./server"}

# 2. Group them
compass.process.group.create {"name": "Full Stack", "processIds": ["frontend-id", "backend-id"]}

# 3. Start all at once
compass.process.group.start {"id": "group-id"}
```

### Build Process Monitoring
```bash
# 1. Create build watcher
compass.process.create {"name": "Build", "command": "npm", "args": ["run", "build:watch"], "type": "build-tool"}

# 2. Monitor for errors
compass.process.logs {"id": "build-id", "limit": 50}
```

## Error Handling

- **Process won't start**: Check working directory, command path, permissions
- **Process crashes immediately**: Review logs for error messages, check dependencies
- **Can't connect to port**: Verify port isn't already in use, check firewall
- **Logs not appearing**: Ensure process outputs to stdout/stderr, not log files
- **Permission denied**: Check file permissions, working directory access

## Best Practices

1. **Always specify process type** for better categorization
2. **Use meaningful names** that describe the process purpose
3. **Set working directories** explicitly for multi-project setups
4. **Monitor logs regularly** during development
5. **Use process groups** for related services
6. **Clean up stopped processes** periodically with `compass.process.list`