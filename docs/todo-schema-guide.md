# TODO Schema Guide - The 3 C's Structure

Compass enforces a structured approach to TODO creation using the **3 C's**: Card, Context, and Criteria. This ensures every task has sufficient information for effective execution.

## Schema Structure

### 1. Card (What needs to be done)
The Card contains the basic task information:
- **title** (required, min 5 chars): Clear, actionable task title
- **description** (required, min 20 chars): Detailed description
- **priority**: low, medium, or high
- **dueDate**: ISO 8601 date-time format
- **estimatedHours**: Estimated time to complete
- **labels**: Array of categorization tags
- **assignedTo**: Person responsible

### 2. Context (Where and with what)
The Context provides environmental information:
- **files** (min 1 item): Files involved in this task
- **dependencies** (required): Task IDs or external dependencies
- **assumptions** (required, min 1 item): Assumptions being made

### 3. Criteria (How to know it's done)
The Criteria defines completion:
- **acceptance** (required, min 2 items): Clear acceptance criteria
- **verification**: How to verify criteria are met

## Example Usage

```json
{
  "card": {
    "title": "Implement user authentication",
    "description": "Add secure user authentication with JWT tokens and proper session management",
    "priority": "high",
    "estimatedHours": 8,
    "labels": ["security", "backend"],
    "assignedTo": "john.doe"
  },
  "context": {
    "files": [
      "src/auth/login.ts",
      "src/middleware/auth.ts",
      "src/models/user.ts"
    ],
    "dependencies": ["database-setup-task-id"],
    "assumptions": [
      "Using JWT for token management",
      "PostgreSQL is the database",
      "bcrypt for password hashing"
    ]
  },
  "criteria": {
    "acceptance": [
      "Users can register with email and password",
      "Users can login and receive a JWT token",
      "Protected routes require valid JWT",
      "Tokens expire after 24 hours"
    ],
    "verification": [
      "Run auth integration tests",
      "Manual testing with Postman",
      "Security audit with OWASP checklist"
    ]
  }
}
```

## Benefits of the 3 C's Structure

1. **Clarity**: Every task has a clear description and success criteria
2. **Context Preservation**: Dependencies and assumptions are documented
3. **Verifiability**: Clear acceptance criteria make completion objective
4. **AI-Friendly**: Structured data helps AI assistants understand tasks better
5. **Reduced Ambiguity**: Minimum requirements prevent vague task definitions

## Common Patterns

### Feature Implementation
```json
{
  "card": {
    "title": "Add user profile page",
    "description": "Create a user profile page showing user details, activity history, and settings"
  },
  "context": {
    "files": ["src/pages/profile.tsx", "src/api/user.ts"],
    "dependencies": ["auth-system", "user-api"],
    "assumptions": ["React for frontend", "Existing user API"]
  },
  "criteria": {
    "acceptance": [
      "Profile page displays user info",
      "Activity history is paginated",
      "Settings can be updated"
    ]
  }
}
```

### Bug Fix
```json
{
  "card": {
    "title": "Fix memory leak in chat component",
    "description": "Chat component doesn't clean up WebSocket listeners causing memory leak on unmount"
  },
  "context": {
    "files": ["src/components/Chat.tsx"],
    "dependencies": [],
    "assumptions": ["WebSocket connection exists", "React useEffect for lifecycle"]
  },
  "criteria": {
    "acceptance": [
      "WebSocket listeners cleaned up on unmount",
      "Memory usage stable after multiple mounts/unmounts"
    ],
    "verification": ["Chrome DevTools memory profiling"]
  }
}
```

### Documentation
```json
{
  "card": {
    "title": "Document API endpoints",
    "description": "Create comprehensive API documentation including request/response examples and error codes"
  },
  "context": {
    "files": ["docs/api.md", "src/routes/*.ts"],
    "dependencies": ["api-implementation"],
    "assumptions": ["OpenAPI format", "All endpoints are stable"]
  },
  "criteria": {
    "acceptance": [
      "All endpoints documented with examples",
      "Error codes and responses included",
      "Authentication requirements specified"
    ]
  }
}
```

## Validation Rules

The schema enforces these validation rules:

1. **Card**:
   - Title must be at least 5 characters
   - Description must be at least 20 characters
   - Priority must be one of: low, medium, high

2. **Context**:
   - At least 1 file must be specified
   - Dependencies array is required (can be empty)
   - At least 1 assumption must be provided

3. **Criteria**:
   - At least 2 acceptance criteria required
   - Each criterion should be specific and measurable

## Tips for Writing Good TODOs

1. **Be Specific**: Instead of "Fix bug", write "Fix null pointer exception in user service when email is empty"

2. **Include Context**: Don't assume future context, document current understanding

3. **Make Criteria Measurable**: Instead of "Make it fast", write "Response time under 200ms for 95% of requests"

4. **List Real Files**: Include actual file paths that will be modified

5. **Document Assumptions**: What might change that would affect this task?

6. **Think Verification**: How will you prove this task is complete?

This structured approach ensures tasks are well-defined, reducing miscommunication and improving development efficiency.