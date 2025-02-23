package main

import (
	"bufio"
	"fmt"
	"os"

	"aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

func main() {
	// Initialize zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	fmt.Println("Welcome to the AIAgent Console Application!")

	// Example usage of an entity
	agent := entities.NewAgent("ConsoleAgent", "http://example.com", "gpt-3.5-turbo", "dummy-key", "Hello, AI Agent!")
	fmt.Printf("Agent Prompt: %s\n", agent.SystemPrompt)

	// Interactive loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("You: ")
		input, _ := reader.ReadString('\n')
		logger.Info("User input received", zap.String("input", input))

		// Simulate AI agent response
		response := "AI Agent: " + agent.SystemPrompt
		fmt.Println(response)
		logger.Info("AI agent response", zap.String("response", response))
	}
}
