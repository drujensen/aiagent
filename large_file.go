package main

import "fmt"

func main() {
    fmt.Println("This is a large test file.")
    for i := 0; i < 100; i++ {
        fmt.Println(fmt.Sprintf("Line %d", i+1))
    }
}