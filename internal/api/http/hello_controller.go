package http

import (
    "net/http"
    "fmt"
)

// HelloController handles HTTP requests for the hello endpoint
func HelloController(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Hello, World!")
}
