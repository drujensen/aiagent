package entities

type Tool struct {
    ID           string
    Name         string
    Type         string
    Configuration map[string]interface{}
}
