package main

import (
    "net/http"

    customhttp "aiagent/internal/api/http"
    "go.uber.org/zap"
)

func main() {
    // Initialize zap logger
    logger, _ := zap.NewProduction()
    defer logger.Sync() // flushes buffer, if any

    logger.Info("Starting HTTP server on :8080")

    http.HandleFunc("/hello", customhttp.HelloController)
    if err := http.ListenAndServe(":8080", nil); err != nil {
        logger.Fatal("Failed to start server", zap.Error(err))
    }
}
