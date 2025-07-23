# Quick TODO Creation Guide

This guide explains the new simplified TODO creation feature in Compass MCP, designed to improve developer workflow by allowing fast task creation without the full 3 C's structure.

## Overview

Compass now supports two ways to create TODOs:

1. **Full TODO** (`compass.todo.create`) - Complete 3 C's structure (Card, Context, Criteria)
2. **Quick TODO** (`compass.todo.quick`) - Simplified creation for fast task entry

## Quick TODO Creation

The `compass.todo.quick` command allows developers to quickly create tasks with minimal required information, while automatically generating sensible defaults for the full structure.

### Basic Usage

```json
compass.todo.quick {
  "title": "Fix login bug",
  "description": "Users can't log in with email addresses"
}
```

### Full Parameters

```json
compass.todo.quick {
  "projectId": "optional-project-id",  // Uses current project if omitted
  "title": "Implement user dashboard",
  "description": "Create a dashboard showing user analytics",
  "priority": "high",                   // low, medium, high
  "dueDate": "2024-12-31T23:59:59Z",
  "labels": ["frontend", "ui"],
  "assignedTo": "john.doe"
}
```

## What Happens Behind the Scenes

When you create a quick TODO, the system automatically generates:

### Card (Task Information)
- **Title**: Your provided title
- **Description**: Your description or "Quick todo item - details to be added later"
- **Priority**: Your priority or "medium" (default)
- **Due Date**: Your date or null
- **Labels**: Your labels or empty array
- **Assigned To**: Your assignee or null
- **Estimated Hours**: null (can be updated later)

### Context (Where and With What)
- **Files**: Empty array (can be added later)
- **Dependencies**: Empty array (can be added later)
- **Assumptions**: ["This is a quick todo - assumptions to be refined as needed"]

### Criteria (How to Know It's Done)
- **Acceptance**: 
  - "Task completed as described in title"
  - "Implementation meets basic requirements"
- **Verification**: ["Manual verification of completion"]

## When to Use Quick vs Full TODOs

### Use Quick TODOs For:
- ✅ Bug reports and fixes
- ✅ Small feature requests
- ✅ Maintenance tasks
- ✅ Quick notes and reminders
- ✅ Brainstorming and idea capture
- ✅ When you want to capture an idea quickly during development

### Use Full TODOs For:
- ✅ Complex features requiring specific context
- ✅ Tasks with multiple dependencies
- ✅ Integration tasks requiring specific files
- ✅ Tasks with detailed acceptance criteria
- ✅ Critical features needing comprehensive documentation

## Examples

### Development Workflow

```bash
# Quick bug report
compass.todo.quick {
  "title": "Fix responsive layout on mobile",
  "description": "Header overlaps content on screens < 768px",
  "priority": "high",
  "labels": ["bug", "css", "mobile"]
}

# Quick feature idea
compass.todo.quick {
  "title": "Add dark mode toggle",
  "labels": ["enhancement", "ui"]
}

# Quick maintenance task
compass.todo.quick {
  "title": "Update dependencies",
  "description": "npm audit shows 3 vulnerabilities",
  "priority": "medium",
  "labels": ["maintenance", "security"]
}
```

### Converting Quick TODOs

You can later convert a quick TODO to a full TODO by updating it with additional context:

```bash
# First, get the TODO details
compass.todo.list {"limit": 1}

# Then update with full context using compass.task.update
compass.task.update {
  "id": "todo-id",
  "updates": {
    "files": ["components/Header.tsx", "styles/mobile.css"],
    "dependencies": ["responsive-design-system-todo"],
    "acceptance": [
      "Header displays correctly on all mobile devices",
      "No content overlap on screens 320px-768px",
      "Navigation remains functional on mobile"
    ]
  }
}
```

## Best Practices

### 1. Use Descriptive Titles
```bash
# Good
"Fix user authentication redirect loop"

# Less Good  
"Fix auth bug"
```

### 2. Add Context in Description
```bash
# Good
{
  "title": "Optimize database queries",
  "description": "User dashboard loads slowly due to N+1 queries on the posts table"
}

# Less Good
{
  "title": "Optimize database queries",
  "description": "Performance issue"
}
```

### 3. Use Labels for Organization
```bash
{
  "title": "Add email validation",
  "labels": ["frontend", "validation", "forms", "user-experience"]
}
```

### 4. Set Appropriate Priorities
- **high**: Bugs, security issues, blockers
- **medium**: Features, improvements (default)
- **low**: Nice-to-have features, cleanup tasks

### 5. Assign Due Dates for Time-Sensitive Tasks
```bash
{
  "title": "Update SSL certificates",
  "dueDate": "2024-12-15T00:00:00Z",
  "priority": "high",
  "labels": ["security", "maintenance"]
}
```

## Integration with Other Features

### With Process Management
Quick TODOs work seamlessly with process management:

```bash
# Create a TODO for starting a development server
compass.todo.quick {
  "title": "Start React development server",
  "description": "Use react-dev template for quick setup"
}

# Then use process templates
compass.process.create {
  "template": "react-dev",
  "name": "React Dev Server"
}
```

### With Project Intelligence
Quick TODOs appear in project summaries and analytics:

```bash
compass.project.summary {}
# Will include quick TODOs in:
# - Task statistics
# - Recent activity
# - Recommendations
```

## Error Handling

The quick TODO system includes enhanced error handling:

```bash
# Missing title
compass.todo.quick {"description": "Fix something"}
# Error: "title is required"

# Invalid priority
compass.todo.quick {
  "title": "Fix bug",
  "priority": "urgent"
}
# Error: Priority must be one of: low, medium, high

# Invalid due date format
compass.todo.quick {
  "title": "Fix bug",
  "dueDate": "tomorrow"
}
# Error: Invalid date format, use ISO 8601 (YYYY-MM-DDTHH:MM:SSZ)
```

## Migration from Other Systems

### From GitHub Issues
```bash
compass.todo.quick {
  "title": "Issue #123: Add user profile pictures",
  "description": "Allow users to upload and display profile images",
  "labels": ["enhancement", "user-profile"],
  "assignedTo": "developer-username"
}
```

### From Jira Tickets
```bash
compass.todo.quick {
  "title": "PROJ-456: Implement OAuth login",
  "description": "Add Google and GitHub OAuth login options",
  "priority": "high",
  "labels": ["authentication", "oauth", "security"]
}
```

### From Linear Tasks
```bash
compass.todo.quick {
  "title": "LIN-789: Redesign onboarding flow",
  "description": "Simplify the 5-step onboarding to 3 steps",
  "labels": ["onboarding", "ux", "frontend"],
  "dueDate": "2024-12-20T00:00:00Z"
}
```

This simplified TODO creation significantly improves the developer experience while maintaining the power and structure of the full Compass task management system.