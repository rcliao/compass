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

	// Initialize MCP server
	mcpServer := mcp.NewMCPServer(taskService, projectService)

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
	fmt.Println("Example usage:")
	fmt.Println("  compass.project.create {\"name\":\"My Project\",\"description\":\"A test project\",\"goal\":\"Learn Compass\"}")
	fmt.Println("  compass.task.create {\"projectId\":\"<project-id>\",\"title\":\"Setup\",\"description\":\"Initial setup\"}")
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