package http

import (
    "net/http"
    "fmt"

    "aiagent/internal/domain/entities"
)

// HelloController handles HTTP requests for the hello endpoint
func HelloController(w http.ResponseWriter, r *http.Request) {
    agent := entities.AIAgent{
        Prompt: "Hello from the HTTP Controller!",
        Tools:  nil, // Add tools as needed
    }

    fmt.Fprintf(w, "Agent Prompt: %s", agent.Prompt)
}
