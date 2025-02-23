package tools

import (
	"bytes"
	"fmt"
	"os/exec"
)

type BashTool struct {
	workspace string
}

func NewBashTool(workspace string) *BashTool {
	return &BashTool{workspace: workspace}
}

func (t *BashTool) Name() string {
	return "Bash"
}

func (t *BashTool) Execute(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("bash command cannot be empty")
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = t.workspace

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
