# Compass MCP Integration Guide

This guide provides step-by-step instructions for integrating Compass with Claude Code as an MCP (Model Context Protocol) server, along with testing protocols to evaluate its effectiveness.

## Overview

Compass serves as an MCP server that provides:
- **Context Management**: Persistent project context across coding sessions
- **Task/TODO Management**: Integrated task tracking with planning features  
- **Process Management**: Run and monitor development servers, build tools, and tests
- **Planning Integration**: Structured planning sessions with discovery tracking

## Prerequisites

- Go 1.21+ installed
- Claude Code installed and configured
- Basic understanding of MCP concepts

## Phase 1: Setup and Configuration

### 1.1 Build Compass MCP Server

```bash
# Navigate to the Compass project directory
cd /path/to/compass

# Build the Compass binary
go build -o bin/compass cmd/compass/main.go

# Make it executable
chmod +x bin/compass

# Test the basic functionality
./bin/compass
```

### 1.2 Configure Claude Code MCP Integration

Add Compass as an MCP server to Claude Code:

```bash
# Add Compass as a stdio MCP server (use absolute path)
claude mcp add compass /path/to/compass/bin/compass --mcp

# Verify the server is registered
claude mcp list

# Check server status
claude mcp status compass
```

**Important**: Make sure to use the absolute path to the Compass binary and include the `--mcp` flag to enable MCP transport mode.

### 1.3 Initialize Compass Project

Create your first project to test the integration:

```bash
# Start Compass CLI to test basic functionality
./bin/compass

# In the Compass CLI, create a project:
compass.project.create {"name":"Test Project","description":"Testing Compass MCP integration","goal":"Evaluate MCP effectiveness"}

# Set it as current project:
compass.project.set_current {"id":"<project-id>"}
```

## Phase 2: Usage Workflows

### 2.1 Context Management Workflow

**Scenario**: Maintaining project context across coding sessions

```markdown
1. **Session Start**:
   - Ask Claude: "What's my current project context using @compass?"
   - Claude will query: compass.project.current, compass.context.get

2. **During Development**:
   - "Record discovery: Users need OAuth integration @compass"
   - "Mark current task as blocked due to API rate limits @compass"
   - "Add decision: Using JWT for session management @compass"

3. **Context Retrieval**:
   - "What files am I working on for authentication? @compass"
   - "Show me related tasks for the login feature @compass"
```

**Expected MCP Commands Used**:
- `compass.context.get` - Retrieve task context
- `compass.context.search` - Search for related information  
- `compass.discovery.add` - Record insights
- `compass.decision.record` - Document decisions

### 2.2 Process Management Workflow

**Scenario**: Managing development servers and tools

```markdown
1. **Start Development Environment**:
   - "Start my web server using @compass process manager"
   - Creates and starts a web server process
   - Captures logs and monitors health

2. **Monitor and Debug**:
   - "Show me the server logs @compass"
   - "What processes are currently running @compass?"
   - "Restart the API server @compass"

3. **Development Workflow**:
   - "Stop all processes before switching branches @compass"
   - "Start the test suite in the background @compass"
```

**Expected MCP Commands Used**:
- `compass.process.create` - Define new processes
- `compass.process.start/stop` - Control process lifecycle
- `compass.process.logs` - Access process output
- `compass.process.list` - View running processes

### 2.3 TODO Management Workflow

**Scenario**: Integrated task tracking during coding

```markdown
1. **Task Creation**:
   - "Add TODO: Implement user registration with email validation @compass"
   - "Set high priority on database migration TODO @compass"
   - "Add labels 'backend' and 'security' to auth TODO @compass"

2. **Progress Tracking**:
   - "Mark authentication TODO as completed @compass"
   - "Update progress: spent 2 hours on API integration @compass"
   - "Show my overdue TODOs @compass"

3. **Organization**:
   - "List all high-priority TODOs @compass"
   - "Show TODOs due this week @compass"
   - "Set due date for deployment TODO to Friday @compass"
```

**Expected MCP Commands Used**:
- `compass.todo.create` - Create new TODO items
- `compass.todo.complete` - Mark items as done
- `compass.todo.list` - Filter and view TODOs
- `compass.todo.overdue` - Find overdue items
- `compass.todo.priority` - Manage priorities

### 2.4 Planning Integration Workflow

**Scenario**: End-to-end feature planning and implementation

```markdown
1. **Planning Phase**:
   - "Start a planning session for user authentication feature @compass"
   - Break down feature into tasks with Compass tracking
   - Record assumptions and dependencies

2. **Implementation Phase**:
   - Use Compass context to maintain awareness of decisions
   - Track discoveries as development reveals new requirements
   - Update task progress and blockers

3. **Review Phase**:
   - "Generate project summary @compass"
   - Review decisions and discoveries made
   - Plan next iteration based on learnings
```

**Expected MCP Commands Used**:
- `compass.planning.start` - Begin planning sessions
- `compass.task.create` - Break down features into tasks
- `compass.project.summary` - Generate insights

## Phase 3: Testing Experiments

### 3.1 Context Retrieval Effectiveness Test

**Objective**: Measure how well Compass helps Claude Code maintain project context

**Test Protocol**:
1. Work on a multi-file feature implementation for 2 hours
2. Use Compass to track files, dependencies, and decisions
3. Close Claude Code session completely
4. Reopen Claude Code the next day
5. Ask: "What was I working on yesterday? @compass"

**Success Metrics**:
- **Context Accuracy**: How accurately Claude recalls previous work (1-5 scale)
- **Context Completeness**: Percentage of important details remembered
- **Time to Context**: How quickly Claude gets back up to speed

**Baseline Comparison**: Repeat the same test without Compass integration

### 3.2 Process Management Utility Test

**Objective**: Evaluate if Compass process management improves development workflow

**Test Protocol**:
1. Set up a typical development environment (web server, API, database, tests)
2. Use Compass to manage all processes for one week
3. Track time spent on process management tasks
4. Note any process-related issues or benefits

**Success Metrics**:
- **Time Savings**: Minutes saved per day on process management
- **Reliability**: Number of process-related issues encountered
- **Convenience**: Subjective rating of workflow improvement (1-5 scale)

**Test Scenarios**:
- Starting full development environment
- Debugging server issues through log access
- Switching between different development modes

### 3.3 TODO Integration Effectiveness Test

**Objective**: Test integrated TODO management vs external tools

**Test Protocol**:
1. Use Compass TODO management for one sprint/week
2. Track task completion rates and context switches
3. Compare against previous workflows with external TODO tools
4. Measure integration friction and benefits

**Success Metrics**:
- **Task Completion Rate**: Percentage of TODOs completed on time
- **Context Retention**: How well context is maintained between tasks
- **Integration Friction**: Time spent switching between tools

**Test Data to Collect**:
- Number of TODOs created, completed, overdue
- Time spent in TODO management vs actual coding
- Instances where TODO context helped with implementation

### 3.4 End-to-End Planning Flow Test

**Objective**: Evaluate complete planning-to-implementation workflow

**Test Protocol**:
1. Start with a high-level feature request
2. Use Compass planning session to break down work
3. Implement feature with Claude Code assistance
4. Track all discoveries and decisions through Compass
5. Generate final project summary

**Success Metrics**:
- **Planning Accuracy**: How well initial planning matched implementation
- **Decision Tracking**: Completeness of decision capture
- **Discovery Rate**: Number of insights captured during development
- **Context Continuity**: How well context was maintained throughout

## Phase 4: Evaluation Framework

### 4.1 Daily Usage Log

Track daily usage with this template:

```markdown
Date: [DATE]
Session Duration: [DURATION]
Project: [PROJECT_NAME]

MCP Commands Used:
- compass.project.*: [COUNT]
- compass.task.*: [COUNT] 
- compass.context.*: [COUNT]
- compass.process.*: [COUNT]
- compass.todo.*: [COUNT]

Effectiveness Ratings (1-5):
- Context Retrieval: [ ]
- Process Management: [ ]
- TODO Tracking: [ ]
- Planning Integration: [ ]

Benefits Observed:
- 

Pain Points:
- 

Time Saved: [ESTIMATE]
Time Lost: [ESTIMATE]

Would use again: Y/N
Most valuable feature: [FEATURE]
Least valuable feature: [FEATURE]
```

### 4.2 Weekly Review Questions

Answer these questions each week:

1. **Context Management**:
   - How often did Compass help you recall previous work?
   - Were you able to pick up where you left off more easily?
   - How accurate was the context information?

2. **Process Management**:
   - Did process management through Compass feel natural?
   - How often did you encounter process-related issues?
   - Would you prefer this over manual process management?

3. **TODO Integration**:
   - How well did TODO integration fit your workflow?
   - Did you complete more tasks when using Compass?
   - Was the TODO context helpful during implementation?

4. **Overall Integration**:
   - How seamless was the MCP integration?
   - Did you encounter any technical issues?
   - What features would make this more valuable?

### 4.3 Quantitative Metrics

Track these metrics over time:

**Productivity Metrics**:
- Tasks completed per day/week
- Time spent on context switching
- Number of "lost context" incidents
- Development environment setup time

**Quality Metrics**:
- Number of decisions documented
- Discovery capture rate
- Context accuracy scores
- Planning vs implementation variance

**Usage Metrics**:
- MCP commands per session
- Most/least used features
- Error rates and friction points
- Session duration and frequency

## Phase 5: Troubleshooting Guide

### 5.1 Common MCP Integration Issues

**Issue**: Compass MCP server not responding
```bash
# Check if process is running
ps aux | grep compass

# Check MCP server logs
claude mcp logs compass

# Restart the server
claude mcp restart compass
```

**Issue**: Commands timing out
```bash
# Check server configuration
claude mcp status compass

# Verify file permissions
ls -la bin/compass

# Test basic connectivity
./bin/compass help
```

**Issue**: Context not updating
```bash
# Verify project is set
compass.project.current

# Check file permissions in .compass directory
ls -la .compass/

# Clear any cached data
rm -rf .compass/cache
```

### 5.2 Performance Issues

**Issue**: Slow MCP responses
- Check disk space in .compass directory
- Verify no large log files accumulating
- Consider process log rotation settings

**Issue**: Memory usage concerns
- Monitor process memory usage
- Check for log buffer size settings
- Review process health check intervals

### 5.3 Data Issues

**Issue**: Lost project data
- Check .compass directory structure
- Verify JSON file integrity
- Review backup strategies

**Issue**: Inconsistent state
- Restart Compass MCP server
- Verify project relationships
- Check for orphaned records

## Phase 6: Iteration and Improvement

Based on testing results, prioritize improvements:

### High-Impact Improvements
1. **Context Search Enhancement**: Better fuzzy search and semantic matching
2. **Process Health Monitoring**: Advanced health checks and auto-restart
3. **TODO Smart Prioritization**: AI-assisted priority and dependency management
4. **Planning Template System**: Reusable planning templates for common workflows

### Medium-Impact Improvements
1. **Integration with Git**: Automatic context updates from commits
2. **Time Tracking**: Automatic time tracking for tasks and processes
3. **Notification System**: Alerts for overdue tasks and process failures
4. **Export/Import**: Integration with external tools and services

### Low-Impact Improvements
1. **UI Enhancements**: Better CLI formatting and colors
2. **Documentation**: More comprehensive help and examples
3. **Logging**: Enhanced logging for debugging and analytics
4. **Configuration**: More customizable settings and preferences

## Conclusion

This integration guide provides a comprehensive framework for evaluating Compass as an MCP server with Claude Code. The key to successful evaluation is:

1. **Systematic Testing**: Follow the structured test protocols
2. **Consistent Measurement**: Use the evaluation framework consistently
3. **Honest Assessment**: Track both benefits and friction points
4. **Iterative Improvement**: Use findings to guide development priorities

The goal is to determine whether Compass MCP integration provides measurable value in real-world coding workflows, and to identify the most impactful features for future development.