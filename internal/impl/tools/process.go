package tools

import (
	"bytes"
	"encoding/json"
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
	configuration map[string]string // Includes "command" (e.g., "git") and "workspace"
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
		return "", nil
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		t.logger.Error("Workspace configuration is missing")
		return "", nil
	}

	if args.Action == "" {
		args.Action = "run"
	}

	switch args.Action {
	case "run":
		return t.runCommand(baseCommand, args, workspace)
	case "status":
		return t.checkStatus(args.PID)
	case "kill":
		return t.killProcess(args.PID)
	default:
		t.logger.Error("Unknown action", zap.String("action", args.Action))
		return "", nil
	}
}

func (t *ProcessTool) runCommand(baseCommand string, args ProcessArgs, workspace string) (string, error) {
	// Split the base command and arguments
	cmdArgs := []string{baseCommand}
	if args.Arguments != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args.Arguments)...)
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
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
				zap.String("arguments", args.Arguments),
				zap.Error(err))
			return "", err
		}
		pid := cmd.Process.Pid
		t.processes[pid] = cmd
		t.logger.Info("Background command started",
			zap.String("command", baseCommand),
			zap.String("arguments", args.Arguments),
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
					zap.String("arguments", args.Arguments),
					zap.Error(err),
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
				zap.String("arguments", args.Arguments))
			return t.toJSON(resp)
		case <-timer.C:
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			t.logger.Warn("Command timed out",
				zap.String("command", baseCommand),
				zap.String("arguments", args.Arguments),
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
			zap.String("arguments", args.Arguments),
			zap.Error(err),
			zap.String("stderr", stderr.String()))
	} else {
		resp.Status = "completed"
		t.logger.Info("Command executed successfully",
			zap.String("command", baseCommand),
			zap.String("arguments", args.Arguments))
	}
	return t.toJSON(resp)
}

func (t *ProcessTool) checkStatus(pid int) (string, error) {
	if pid == 0 {
		t.logger.Error("PID is required for status check")
		return "", nil
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
		return "", nil
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
