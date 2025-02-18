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
    agent := entities.AIAgent{
        Prompt: "Hello, AI Agent!",
        Tools:  nil, // Add tools as needed
    }

    fmt.Printf("Agent Prompt: %s\n", agent.Prompt)

    // Interactive loop
    reader := bufio.NewReader(os.Stdin)
    for {
        fmt.Print("You: ")
        input, _ := reader.ReadString('\n')
        logger.Info("User input received", zap.String("input", input))

        // Simulate AI agent response
        response := "AI Agent: " + agent.Prompt
        fmt.Println(response)
        logger.Info("AI agent response", zap.String("response", response))
    }
}
