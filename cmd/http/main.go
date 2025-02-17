package main

import (
    "log"
    "net/http"

    customhttp "AIAgent/internal/api/http"
)

func main() {
    http.HandleFunc("/hello", customhttp.HelloController)
    log.Println("Starting HTTP server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
