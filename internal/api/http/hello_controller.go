package http

import (
    "net/http"
    "fmt"

    "aiagent/internal/domain/entities"
    "go.uber.org/zap"
)

// HelloController handles HTTP requests for the hello endpoint
func HelloController(w http.ResponseWriter, r *http.Request) {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    logger.Info("Received HTTP request",
        zap.String("method", r.Method),
        zap.String("url", r.URL.String()),
    )

    agent := entities.AIAgent{
        Prompt: "Hello from the HTTP Controller!",
        Tools:  nil, // Add tools as needed
    }

    fmt.Fprintf(w, "Agent Prompt: %s", agent.Prompt)
}
