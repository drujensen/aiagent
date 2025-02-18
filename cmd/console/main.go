package main

import (
    "fmt"
    "aiagent/internal/domain/entities"
)

func main() {
    fmt.Println("Welcome to the AIAgent Console Application!")

    // Example usage of an entity
    agent := entities.AIAgent{
        Prompt: "Hello, AI Agent!",
        Tools:  nil, // Add tools as needed
    }

    fmt.Printf("Agent Prompt: %s\n", agent.Prompt)
}
