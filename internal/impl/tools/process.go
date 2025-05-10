package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

type ProcessTool struct {
	name          string
	description   string
	configuration map[string]string // Includes "command" (e.g., "git"), "workspace", and "extraArgs"
	logger        *zap.Logger
	processes     map[int]*exec.Cmd // Track background processes by PID
}

func NewProcessTool(name, description string, configuration map[string]string, logger *zap.Logger) *ProcessTool {
	return &ProcessTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		processes:     make(map[int]*exec.Cmd),
	}
}

func (t *ProcessTool) Name() string {
	return t.name
}

func (t *ProcessTool) Description() string {
	return t.description
}

func (t *ProcessTool) Configuration() map[string]string {
	return t.configuration
}

func (t *ProcessTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *ProcessTool) FullDescription() string {
	var b strings.Builder

	// Add description
	b.WriteString(t.Description())
	b.WriteString("\nNote: Output is limited to 4096 tokens (~16,384 characters).\n\n")

	// Add configuration header
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")

	// Loop through configuration and add key-value pairs to the table
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}

	return b.String()
}

func (t *ProcessTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "arguments",
			Type:        "string",
			Description: "Arguments to pass to the configured command (e.g., 'clone https://github.com/repo')",
			Required:    false,
		},
		{
			Name:        "background",
			Type:        "boolean",
			Description: "Run the command in the background (e.g., for web servers)",
			Required:    false,
		},
		{
			Name:        "timeout",
			Type:        "integer",
			Description: "Timeout in seconds (default: 30, 0 for no timeout)",
			Required:    false,
		},
		{
			Name: "env",
			Type: "array",
			Items: []entities.Item{
				{Type: "string"},
			},
			Description: "Environment variables as key=value pairs (e.g., ['PORT=8080'])",
			Required:    false,
		},
		{
			Name:        "pid",
			Type:        "integer",
			Description: "PID of a background process to check or kill",
			Required:    false,
		},
		{
			Name:        "action",
			Type:        "string",
			Enum:        []string{"run", "status", "kill"},
			Description: "Action to perform: run (default), status (check PID), or kill (stop PID)",
			Required:    false,
		},
	}
}

type ProcessResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	PID    int    `json:"pid,omitempty"`
	Status string `json:"status,omitempty"`
}

type ProcessArgs struct {
	Arguments  string   `json:"arguments"`
	Background bool     `json:"background"`
	Timeout    int      `json:"timeout"`
	Env        []string `json:"env"`
	PID        int      `json:"pid"`
	Action     string   `json:"action"`
}

func (t *ProcessTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing process", zap.String("arguments", arguments))

	var args ProcessArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	baseCommand := t.configuration["command"]
	if baseCommand == "" {
		t.logger.Error("Base command not specified in configuration")
		return "", fmt.Errorf("base command not specified in configuration")
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current directory: %v", err)
		}
	}

	if args.Action == "" {
		args.Action = "run"
	}

	switch args.Action {
	case "run":
		results, err := t.runCommand(baseCommand, args, workspace)
		if len(results) > 16384 {
			results = results[:16384] + "...truncated"
		}
		return results, err
	case "status":
		return t.checkStatus(args.PID)
	case "kill":
		return t.killProcess(args.PID)
	default:
		t.logger.Error("Unknown action", zap.String("action", args.Action))
		return "", fmt.Errorf("unknown action: %s", args.Action)
	}
}

// splitShellArgs splits a command string into arguments, respecting quoted strings and escapes
func splitShellArgs(input string) []string {
	var args []string
	var current strings.Builder
	var inQuote bool
	var quoteChar rune
	escaped := false

	input = strings.TrimSpace(input)
	runes := []rune(input)

	for i := range runes {
		ch := runes[i]

		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if inQuote {
			if ch == quoteChar {
				inQuote = false
				quoteChar = 0
				continue
			}
			current.WriteRune(ch)
			continue
		}

		if ch == '"' || ch == '\'' {
			inQuote = true
			quoteChar = ch
			continue
		}

		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func (t *ProcessTool) runCommand(baseCommand string, args ProcessArgs, workspace string) (string, error) {
	// Initialize command arguments
	var cmdArgs []string

	// Add extraArgs from configuration if present
	if extraArgs, exists := t.configuration["extraArgs"]; exists && extraArgs != "" {
		extraArgsParsed := splitShellArgs(extraArgs)
		cmdArgs = append(cmdArgs, extraArgsParsed...)
	}

	// Append arguments from ProcessArgs if provided
	if args.Arguments != "" {
		parsedArgs := splitShellArgs(args.Arguments)
		cmdArgs = append(cmdArgs, parsedArgs...)
	}

	// Create the command with the base command and combined arguments
	cmd := exec.Command(baseCommand, cmdArgs...)
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), args.Env...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if args.Background {
		err := cmd.Start()
		if err != nil {
			t.logger.Error("Failed to start background command",
				zap.String("command", baseCommand),
				zap.Strings("arguments", cmdArgs),
				zap.Error(err))
			return "", err
		}
		pid := cmd.Process.Pid
		t.processes[pid] = cmd
		t.logger.Info("Background command started",
			zap.String("command", baseCommand),
			zap.Strings("arguments", cmdArgs),
			zap.Int("pid", pid))
		resp := ProcessResponse{
			Stdout: "Command started in background",
			PID:    pid,
			Status: "running",
		}
		return t.toJSON(resp)
	}

	if args.Timeout == 0 {
		args.Timeout = 30 // Default timeout of 30 seconds
	}

	if args.Timeout > 0 {
		timer := time.NewTimer(time.Duration(args.Timeout) * time.Second)
		defer timer.Stop()

		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.Run()
		}()

		select {
		case err := <-errChan:
			if err != nil {
				t.logger.Error("Command execution failed",
					zap.String("command", baseCommand),
					zap.Strings("arguments", cmdArgs),
					zap.Error(err),
					zap.String("stdout", out.String()),
					zap.String("stderr", stderr.String()))
				resp := ProcessResponse{
					Stdout: out.String(),
					Stderr: stderr.String(),
					Status: "failed",
				}
				return t.toJSON(resp)
			}
			resp := ProcessResponse{
				Stdout: out.String(),
				Stderr: stderr.String(),
				Status: "completed",
			}
			t.logger.Info("Command executed successfully",
				zap.String("command", baseCommand),
				zap.Strings("arguments", cmdArgs))
			return t.toJSON(resp)
		case <-timer.C:
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			t.logger.Warn("Command timed out",
				zap.String("command", baseCommand),
				zap.Strings("arguments", cmdArgs),
				zap.Int("timeout", args.Timeout))
			resp := ProcessResponse{
				Stdout: out.String(),
				Stderr: stderr.String(),
				Status: "timeout",
			}
			return t.toJSON(resp)
		}
	}

	err := cmd.Run()
	resp := ProcessResponse{
		Stdout: out.String(),
		Stderr: stderr.String(),
	}
	if err != nil {
		resp.Status = "failed"
		t.logger.Error("Command execution failed",
			zap.String("command", baseCommand),
			zap.Strings("arguments", cmdArgs),
			zap.Error(err),
			zap.String("stderr", stderr.String()))
	} else {
		resp.Status = "completed"
		t.logger.Info("Command executed successfully",
			zap.String("command", baseCommand),
			zap.Strings("arguments", cmdArgs))
	}
	return t.toJSON(resp)
}

func (t *ProcessTool) checkStatus(pid int) (string, error) {
	if pid == 0 {
		t.logger.Error("PID is required for status check")
		return "", fmt.Errorf("PID is required for status check")
	}
	cmd, exists := t.processes[pid]
	if !exists {
		resp := ProcessResponse{
			Status: "not found",
		}
		return t.toJSON(resp)
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		delete(t.processes, pid)
		resp := ProcessResponse{
			PID:    pid,
			Status: "exited",
		}
		t.logger.Info("Background process has exited", zap.Int("pid", pid))
		return t.toJSON(resp)
	}
	resp := ProcessResponse{
		PID:    pid,
		Status: "running",
	}
	t.logger.Info("Background process status checked", zap.Int("pid", pid))
	return t.toJSON(resp)
}

func (t *ProcessTool) killProcess(pid int) (string, error) {
	if pid == 0 {
		t.logger.Error("PID is required for kill")
		return "", fmt.Errorf("PID is required for kill")
	}
	cmd, exists := t.processes[pid]
	if !exists {
		resp := ProcessResponse{
			Status: "not found",
		}
		return t.toJSON(resp)
	}
	err := cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.logger.Error("Failed to terminate process",
			zap.Int("pid", pid),
			zap.Error(err))
		return "", err
	}
	delete(t.processes, pid)
	resp := ProcessResponse{
		PID:    pid,
		Status: "terminated",
	}
	t.logger.Info("Background process terminated", zap.Int("pid", pid))
	return t.toJSON(resp)
}

func (t *ProcessTool) toJSON(resp ProcessResponse) (string, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", err
	}
	return string(data), nil
}

var _ entities.Tool = (*ProcessTool)(nil)
