package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/rcliao/compass/internal/domain"
)

// FormatTodosAsMarkdown formats a list of tasks as markdown
func FormatTodosAsMarkdown(todos []*domain.Task) string {
	if len(todos) == 0 {
		return "📋 **No TODOs found**\n\nCreate a new TODO with `compass.todo.create`"
	}

	var sb strings.Builder
	sb.WriteString("# 📋 TODO List\n\n")

	// Group todos by status
	statusGroups := map[domain.TaskStatus][]*domain.Task{
		domain.StatusPlanned:    {},
		domain.StatusInProgress: {},
		domain.StatusCompleted:  {},
		domain.StatusBlocked:    {},
	}

	for _, todo := range todos {
		statusGroups[todo.Card.Status] = append(statusGroups[todo.Card.Status], todo)
	}

	// Display in order: In Progress, Planned, Blocked, Completed
	displayOrder := []domain.TaskStatus{
		domain.StatusInProgress,
		domain.StatusPlanned,
		domain.StatusBlocked,
		domain.StatusCompleted,
	}

	for _, status := range displayOrder {
		tasks := statusGroups[status]
		if len(tasks) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", getStatusHeader(status)))

		for _, task := range tasks {
			sb.WriteString(formatSingleTodo(task))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

// FormatSingleTodoAsMarkdown formats a single task as markdown
func FormatSingleTodoAsMarkdown(todo *domain.Task) string {
	var sb strings.Builder
	sb.WriteString("# 📋 TODO Details\n\n")
	sb.WriteString(formatSingleTodo(todo))
	
	// Add detailed information

	if len(todo.Context.Files) > 0 {
		sb.WriteString(fmt.Sprintf("\n### Files\n- %s\n", strings.Join(todo.Context.Files, "\n- ")))
	}

	if len(todo.Context.Dependencies) > 0 {
		sb.WriteString(fmt.Sprintf("\n### Dependencies\n- %s\n", strings.Join(todo.Context.Dependencies, "\n- ")))
	}

	if len(todo.Criteria.Acceptance) > 0 {
		sb.WriteString(fmt.Sprintf("\n### Acceptance Criteria\n- %s\n", strings.Join(todo.Criteria.Acceptance, "\n- ")))
	}

	return strings.TrimSpace(sb.String())
}

func formatSingleTodo(task *domain.Task) string {
	var sb strings.Builder

	// Status checkbox
	checkbox := "[ ]"
	if task.Card.Status == domain.StatusCompleted {
		checkbox = "[x]"
	} else if task.Card.Status == domain.StatusBlocked {
		checkbox = "[!]"
	} else if task.Card.Status == domain.StatusInProgress {
		checkbox = "[>]"
	}

	// Priority indicator
	priority := ""
	switch task.Card.Priority {
	case domain.PriorityHigh:
		priority = "🔴"
	case domain.PriorityMedium:
		priority = "🟡"
	case domain.PriorityLow:
		priority = "🟢"
	}

	// Basic info
	sb.WriteString(fmt.Sprintf("### %s %s **%s**", checkbox, priority, task.Card.Title))

	// ID (shortened for display)
	if len(task.ID) > 8 {
		sb.WriteString(fmt.Sprintf(" `[%s]`", task.ID[:8]))
	}

	sb.WriteString("\n")

	// Description
	if task.Card.Description != "" {
		sb.WriteString(fmt.Sprintf("   %s\n", task.Card.Description))
	}

	// Due date and overdue indicator
	if task.Card.DueDate != nil {
		dueStr := task.Card.DueDate.Format("Jan 2, 2006")
		if task.IsOverdue() {
			sb.WriteString(fmt.Sprintf("   ⏰ **OVERDUE** (was due %s)\n", dueStr))
		} else {
			daysUntil := int(time.Until(*task.Card.DueDate).Hours() / 24)
			if daysUntil == 0 {
				sb.WriteString(fmt.Sprintf("   ⏰ Due **today**\n"))
			} else if daysUntil == 1 {
				sb.WriteString(fmt.Sprintf("   ⏰ Due **tomorrow**\n"))
			} else {
				sb.WriteString(fmt.Sprintf("   ⏰ Due %s (%d days)\n", dueStr, daysUntil))
			}
		}
	}

	// Labels
	if len(task.Card.Labels) > 0 {
		labels := make([]string, len(task.Card.Labels))
		for i, label := range task.Card.Labels {
			labels[i] = fmt.Sprintf("`%s`", label)
		}
		sb.WriteString(fmt.Sprintf("   🏷️  %s\n", strings.Join(labels, " ")))
	}

	// Progress
	if task.Card.EstimatedHours != nil {
		if task.Card.ActualHours != nil {
			progress := (*task.Card.ActualHours / *task.Card.EstimatedHours) * 100
			sb.WriteString(fmt.Sprintf("   📊 Progress: %.0f%% (%.1fh / %.1fh)\n", 
				progress, *task.Card.ActualHours, *task.Card.EstimatedHours))
		} else {
			sb.WriteString(fmt.Sprintf("   📊 Estimated: %.1fh\n", *task.Card.EstimatedHours))
		}
	}

	// Assigned to
	if task.Card.AssignedTo != nil {
		sb.WriteString(fmt.Sprintf("   👤 Assigned to: %s\n", *task.Card.AssignedTo))
	}

	return sb.String()
}

func getStatusHeader(status domain.TaskStatus) string {
	switch status {
	case domain.StatusPlanned:
		return "📅 Planned"
	case domain.StatusInProgress:
		return "🚀 In Progress"
	case domain.StatusCompleted:
		return "✅ Completed"
	case domain.StatusBlocked:
		return "🚫 Blocked"
	default:
		return string(status)
	}
}

// FormatProjectsAsMarkdown formats a list of projects as markdown
func FormatProjectsAsMarkdown(projects []*domain.Project) string {
	if len(projects) == 0 {
		return "📁 **No projects found**\n\nCreate a new project with `compass.project.create`"
	}

	var sb strings.Builder
	sb.WriteString("# 📁 Projects\n\n")

	for i, project := range projects {
		sb.WriteString(fmt.Sprintf("## %d. %s", i+1, project.Name))
		if len(project.ID) > 8 {
			sb.WriteString(fmt.Sprintf(" `[%s]`", project.ID[:8]))
		}
		sb.WriteString("\n\n")

		if project.Description != "" {
			sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", project.Description))
		}

		if project.Goal != "" {
			sb.WriteString(fmt.Sprintf("**Goal:** %s\n\n", project.Goal))
		}

		sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", project.CreatedAt.Format("Jan 2, 2006")))
		sb.WriteString("---\n\n")
	}

	return strings.TrimSpace(sb.String())
}

// FormatProcessStatusAsMarkdown formats a single process status as markdown
func FormatProcessStatusAsMarkdown(process *domain.Process) string {
	var sb strings.Builder
	sb.WriteString("# 🔄 Process Status\n\n")
	
	// Status emoji
	statusEmoji := getProcessStatusEmoji(process.Status)
	sb.WriteString(fmt.Sprintf("## %s %s", statusEmoji, process.Name))
	if len(process.ID) > 8 {
		sb.WriteString(fmt.Sprintf(" `[%s]`", process.ID[:8]))
	}
	sb.WriteString("\n\n")
	
	// Basic info
	sb.WriteString(fmt.Sprintf("**Status:** %s\n", process.Status))
	sb.WriteString(fmt.Sprintf("**Type:** %s\n", process.Type))
	sb.WriteString(fmt.Sprintf("**Command:** `%s %s`\n", process.Command, strings.Join(process.Args, " ")))
	
	if process.WorkingDir != "" {
		sb.WriteString(fmt.Sprintf("**Working Dir:** %s\n", process.WorkingDir))
	}
	
	if process.PID > 0 {
		sb.WriteString(fmt.Sprintf("**PID:** %d\n", process.PID))
	}
	
	if process.Port > 0 {
		sb.WriteString(fmt.Sprintf("**Port:** %d\n", process.Port))
	}
	
	// Timing info
	if process.StartedAt != nil {
		sb.WriteString(fmt.Sprintf("**Started:** %s\n", process.StartedAt.Format("Jan 2, 15:04:05")))
		if process.IsRunning() {
			duration := process.Duration()
			sb.WriteString(fmt.Sprintf("**Uptime:** %s\n", formatDuration(duration)))
		}
	}
	
	if process.StoppedAt != nil {
		sb.WriteString(fmt.Sprintf("**Stopped:** %s\n", process.StoppedAt.Format("Jan 2, 15:04:05")))
	}
	
	// Health status
	if process.LastHealthCheck != nil {
		sb.WriteString(fmt.Sprintf("\n### Health Check\n"))
		sb.WriteString(fmt.Sprintf("**Last Check:** %s ago\n", formatDuration(time.Since(*process.LastHealthCheck))))
		if process.HealthStatus != "" {
			sb.WriteString(fmt.Sprintf("**Health Status:** %s\n", process.HealthStatus))
		}
	}
	
	// Restart policy
	if process.RestartPolicy.Enabled {
		sb.WriteString(fmt.Sprintf("\n### Restart Policy\n"))
		sb.WriteString(fmt.Sprintf("**Enabled:** Yes\n"))
		sb.WriteString(fmt.Sprintf("**Max Retries:** %d\n", process.RestartPolicy.MaxRetries))
		sb.WriteString(fmt.Sprintf("**Retry Count:** %d\n", process.RestartPolicy.RetryCount))
		if process.RestartPolicy.LastRestart != nil {
			sb.WriteString(fmt.Sprintf("**Last Restart:** %s\n", process.RestartPolicy.LastRestart.Format("Jan 2, 15:04:05")))
		}
	}
	
	// Environment variables
	if len(process.Environment) > 0 {
		sb.WriteString(fmt.Sprintf("\n### Environment Variables\n"))
		for k, v := range process.Environment {
			sb.WriteString(fmt.Sprintf("- `%s=%s`\n", k, v))
		}
	}
	
	return strings.TrimSpace(sb.String())
}

// FormatProcessesAsMarkdown formats a list of processes as markdown
func FormatProcessesAsMarkdown(processes []*domain.Process) string {
	if len(processes) == 0 {
		return "🔄 **No processes found**\n\nCreate a new process with `compass.process.create`"
	}
	
	var sb strings.Builder
	sb.WriteString("# 🔄 Process List\n\n")
	
	// Group by status
	statusGroups := make(map[domain.ProcessStatus][]*domain.Process)
	for _, process := range processes {
		statusGroups[process.Status] = append(statusGroups[process.Status], process)
	}
	
	// Display in order: Running, Starting, Stopping, Pending, Failed/Crashed, Stopped
	displayOrder := []domain.ProcessStatus{
		domain.ProcessStatusRunning,
		domain.ProcessStatusStarting,
		domain.ProcessStatusStopping,
		domain.ProcessStatusPending,
		domain.ProcessStatusFailed,
		domain.ProcessStatusCrashed,
		domain.ProcessStatusStopped,
	}
	
	for _, status := range displayOrder {
		procs := statusGroups[status]
		if len(procs) == 0 {
			continue
		}
		
		sb.WriteString(fmt.Sprintf("## %s %s (%d)\n\n", getProcessStatusEmoji(status), status, len(procs)))
		
		for _, proc := range procs {
			sb.WriteString(formatProcessSummary(proc))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	
	return strings.TrimSpace(sb.String())
}

// FormatProcessLogsAsMarkdown formats process logs as markdown
func FormatProcessLogsAsMarkdown(logs []*domain.ProcessLog) string {
	if len(logs) == 0 {
		return "📋 **No logs found**\n\nThe process may not have produced any output yet."
	}
	
	var sb strings.Builder
	sb.WriteString("# 📋 Process Logs\n\n")
	sb.WriteString("```\n")
	
	for _, log := range logs {
		timestamp := log.Timestamp.Format("15:04:05")
		prefix := ""
		
		switch log.Type {
		case domain.LogTypeStderr:
			prefix = "[ERR]"
		case domain.LogTypeSystem:
			prefix = "[SYS]"
		default:
			prefix = "[OUT]"
		}
		
		sb.WriteString(fmt.Sprintf("%s %s %s\n", timestamp, prefix, log.Message))
	}
	
	sb.WriteString("```\n")
	
	return strings.TrimSpace(sb.String())
}

func formatProcessSummary(process *domain.Process) string {
	var sb strings.Builder
	
	// Status icon and name
	statusEmoji := getProcessStatusEmoji(process.Status)
	sb.WriteString(fmt.Sprintf("### %s **%s**", statusEmoji, process.Name))
	
	// ID (shortened)
	if len(process.ID) > 8 {
		sb.WriteString(fmt.Sprintf(" `[%s]`", process.ID[:8]))
	}
	sb.WriteString("\n")
	
	// Command
	sb.WriteString(fmt.Sprintf("   `%s %s`\n", process.Command, strings.Join(process.Args, " ")))
	
	// Type and port
	sb.WriteString(fmt.Sprintf("   Type: %s", process.Type))
	if process.Port > 0 {
		sb.WriteString(fmt.Sprintf(" | Port: %d", process.Port))
	}
	if process.PID > 0 {
		sb.WriteString(fmt.Sprintf(" | PID: %d", process.PID))
	}
	sb.WriteString("\n")
	
	// Runtime info
	if process.IsRunning() && process.StartedAt != nil {
		uptime := process.Duration()
		sb.WriteString(fmt.Sprintf("   ⏱️  Uptime: %s\n", formatDuration(uptime)))
	}
	
	return sb.String()
}

func getProcessStatusEmoji(status domain.ProcessStatus) string {
	switch status {
	case domain.ProcessStatusRunning:
		return "🟢"
	case domain.ProcessStatusStarting:
		return "🟡"
	case domain.ProcessStatusStopping:
		return "🟠"
	case domain.ProcessStatusPending:
		return "⏸️"
	case domain.ProcessStatusStopped:
		return "⏹️"
	case domain.ProcessStatusFailed:
		return "❌"
	case domain.ProcessStatusCrashed:
		return "💥"
	default:
		return "❓"
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}