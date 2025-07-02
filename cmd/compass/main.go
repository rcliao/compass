package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rcliao/compass/internal/mcp"
	"github.com/rcliao/compass/internal/service"
	"github.com/rcliao/compass/internal/storage"
)

func main() {
	// Get current working directory for file storage
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current directory:", err)
	}

	// Initialize storage
	fileStorage, err := storage.NewFileStorage(cwd)
	if err != nil {
		log.Fatal("Failed to initialize file storage:", err)
	}

	// Initialize services
	taskService := service.NewTaskService(fileStorage)
	projectService := service.NewProjectService(fileStorage)
	contextRetriever := service.NewContextRetriever(fileStorage, fileStorage)
	planningService := service.NewPlanningService(fileStorage, taskService, projectService)
	summaryService := service.NewProjectSummaryService(taskService, projectService, planningService)

	// Initialize MCP server
	mcpServer := mcp.NewMCPServer(taskService, projectService, contextRetriever, planningService, summaryService)

	fmt.Println("Compass MCP Server started")
	fmt.Println("Type 'help' for available commands or 'quit' to exit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("compass> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "quit" || input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if input == "help" {
			printHelp()
			continue
		}

		handleCommand(mcpServer, input)
	}
}

func printHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help                           - Show this help")
	fmt.Println("  quit/exit                      - Exit the application")
	fmt.Println()
	fmt.Println("MCP Commands (JSON format):")
	fmt.Println("  Project commands:")
	fmt.Println("    compass.project.create       - Create a new project")
	fmt.Println("    compass.project.list         - List all projects")
	fmt.Println("    compass.project.current      - Get current project")
	fmt.Println("    compass.project.set_current  - Set current project")
	fmt.Println()
	fmt.Println("  Task commands:")
	fmt.Println("    compass.task.create          - Create a new task")
	fmt.Println("    compass.task.list            - List tasks")
	fmt.Println("    compass.task.get             - Get a specific task")
	fmt.Println("    compass.task.update          - Update a task")
	fmt.Println("    compass.task.delete          - Delete a task")
	fmt.Println()
	fmt.Println("  Context commands:")
	fmt.Println("    compass.context.get          - Get full context for a task")
	fmt.Println("    compass.context.search       - Search tasks by query")
	fmt.Println("    compass.context.check        - Check context sufficiency")
	fmt.Println()
	fmt.Println("  Intelligent queries:")
	fmt.Println("    compass.next                 - Get next recommended task")
	fmt.Println("    compass.blockers             - Get all blocked tasks")
	fmt.Println()
	fmt.Println("  Planning commands:")
	fmt.Println("    compass.planning.start       - Start a new planning session")
	fmt.Println("    compass.planning.list        - List planning sessions")
	fmt.Println("    compass.planning.get         - Get planning session details")
	fmt.Println("    compass.planning.complete    - Complete a planning session")
	fmt.Println("    compass.planning.abort       - Abort a planning session")
	fmt.Println()
	fmt.Println("  Discovery and Decision commands:")
	fmt.Println("    compass.discovery.add        - Record a new discovery")
	fmt.Println("    compass.discovery.list       - List all discoveries")
	fmt.Println("    compass.decision.record      - Record a decision")
	fmt.Println("    compass.decision.list        - List all decisions")
	fmt.Println()
	fmt.Println("  Summary commands:")
	fmt.Println("    compass.project.summary      - Generate intelligent project summary and insights")
	fmt.Println()
	fmt.Println("Example usage:")
	fmt.Println("  compass.project.create {\"name\":\"My Project\",\"description\":\"A test project\",\"goal\":\"Learn Compass\"}")
	fmt.Println("  compass.task.create {\"projectId\":\"<project-id>\",\"title\":\"Setup\",\"description\":\"Initial setup\"}")
	fmt.Println("  compass.context.search {\"query\":\"authentication\",\"limit\":5}")
	fmt.Println("  compass.next {}")
	fmt.Println("  compass.context.get {\"taskId\":\"<task-id>\"}")
	fmt.Println("  compass.planning.start {\"name\":\"Sprint Planning\"}")
	fmt.Println("  compass.discovery.add {\"insight\":\"Users prefer OAuth\",\"impact\":\"high\",\"source\":\"research\"}")
	fmt.Println("  compass.decision.record {\"question\":\"Database choice\",\"choice\":\"PostgreSQL\",\"rationale\":\"Better JSON support\"}")
	fmt.Println("  compass.project.summary {}")

}

func handleCommand(server *mcp.MCPServer, input string) {
	parts := strings.SplitN(input, " ", 2)
	if len(parts) < 1 {
		fmt.Println("Error: Invalid command format")
		return
	}

	method := parts[0]
	var params json.RawMessage

	if len(parts) > 1 {
		paramStr := parts[1]
		if err := json.Unmarshal([]byte(paramStr), &params); err != nil {
			fmt.Printf("Error: Invalid JSON parameters: %v\n", err)
			return
		}
	}

	result, err := server.HandleCommand(method, params)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Pretty print the result
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error formatting result: %v\n", err)
		return
	}

	fmt.Println(string(output))
}