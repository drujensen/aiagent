package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"aiagent/internal/domain/interfaces"
)

type BashTool struct {
	configuration map[string]string
}

func NewBashTool(configuration map[string]string) *BashTool {
	return &BashTool{configuration: configuration}
}

func (t *BashTool) Name() string {
	return "Bash"
}

func (t *BashTool) Description() string {
	return "A tool that executes bash commands"
}

func (t *BashTool) Configuration() []string {
	return []string{
		"workspace",
	}
}

func (t *BashTool) Parameters() []interfaces.Parameter {
	return []interfaces.Parameter{
		{
			Name:        "command",
			Type:        "string",
			Description: "The bash command to execute",
			Required:    true,
		},
	}
}

func (t *BashTool) Execute(arguments string) (string, error) {
	var args struct {
		command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("error parsing tool arguments: %v", err)
	}

	command := args.command
	if command == "" {
		return "", fmt.Errorf("bash command cannot be empty")
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		return "", fmt.Errorf("workspace configuration is required")
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = workspace

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s\n%s", err, stderr.String())
	}

	return out.String(), nil
}
