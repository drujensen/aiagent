package tools

import (
	"bytes"
	"encoding/json"
	"os/exec"

	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type BashTool struct {
	configuration map[string]string
	logger        *zap.Logger
}

func NewBashTool(configuration map[string]string, logger *zap.Logger) *BashTool {
	return &BashTool{
		configuration: configuration,
		logger:        logger,
	}
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
	// Log the arguments being executed
	t.logger.Debug("Executing bash command", zap.String("arguments", arguments))

	var command string
	// Try to unmarshal as a plain string
	if err := json.Unmarshal([]byte(arguments), &command); err != nil {
		// If that fails, try unmarshaling as a JSON object
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			t.logger.Error("Failed to parse arguments", zap.Error(err))
			return "", err
		}
		command = args.Command
	}

	if command == "" {
		t.logger.Error("Command cannot be empty")
		return "", nil
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		t.logger.Error("Workspace configuration is missing")
		return "", nil
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = workspace

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.logger.Error("Bash command execution failed",
			zap.Error(err),
			zap.String("stderr", stderr.String()))
		return "", err
	}

	t.logger.Info("Bash command executed successfully",
		zap.String("output", out.String()))
	return out.String(), nil
}
