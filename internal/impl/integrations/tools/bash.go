package tools

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"syscall"
	"time"

	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type BashTool struct {
	configuration map[string]string
	logger        *zap.Logger
	processes     map[int]*exec.Cmd // Track background processes by PID
}

func NewBashTool(configuration map[string]string, logger *zap.Logger) *BashTool {
	return &BashTool{
		configuration: configuration,
		logger:        logger,
		processes:     make(map[int]*exec.Cmd),
	}
}

func (t *BashTool) Name() string {
	return "Bash"
}

func (t *BashTool) Description() string {
	return "A tool that executes bash commands with support for background processes and full output"
}

func (t *BashTool) Configuration() []string {
	return []string{"workspace"}
}

func (t *BashTool) Parameters() []interfaces.Parameter {
	return []interfaces.Parameter{
		{
			Name:        "command",
			Type:        "string",
			Description: "The bash command to execute",
			Required:    true,
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
			Name:        "env",
			Type:        "array",
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

type BashResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	PID    int    `json:"pid,omitempty"`
	Status string `json:"status,omitempty"`
}

// BashArgs defines the structure of the arguments for bash execution
type BashArgs struct {
	Command    string   `json:"command"`
	Background bool     `json:"background"`
	Timeout    int      `json:"timeout"`
	Env        []string `json:"env"`
	PID        int      `json:"pid"`
	Action     string   `json:"action"`
}

func (t *BashTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing bash command", zap.String("arguments", arguments))

	var args BashArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	if args.Command == "" && args.Action == "" {
		t.logger.Error("Command or action is required")
		return "", nil
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		t.logger.Error("Workspace configuration is missing")
		return "", nil
	}

	// Default action to "run" if not specified
	if args.Action == "" {
		args.Action = "run"
	}

	switch args.Action {
	case "run":
		return t.runCommand(args, workspace)
	case "status":
		return t.checkStatus(args.PID)
	case "kill":
		return t.killProcess(args.PID)
	default:
		t.logger.Error("Unknown action", zap.String("action", args.Action))
		return "", nil
	}
}

func (t *BashTool) runCommand(args BashArgs, workspace string) (string, error) {
	cmd := exec.Command("bash", "-c", args.Command)
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), args.Env...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Handle background processes
	if args.Background {
		err := cmd.Start()
		if err != nil {
			t.logger.Error("Failed to start background command",
				zap.String("command", args.Command),
				zap.Error(err))
			return "", err
		}
		pid := cmd.Process.Pid
		t.processes[pid] = cmd
		t.logger.Info("Background command started",
			zap.String("command", args.Command),
			zap.Int("pid", pid))
		resp := BashResponse{
			Stdout: "Command started in background",
			PID:    pid,
			Status: "running",
		}
		return t.toJSON(resp)
	}

	// Set default timeout if not specified
	if args.Timeout == 0 {
		args.Timeout = 30 // Default timeout of 30 seconds
	}

	// Run with timeout if specified
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
				t.logger.Error("Bash command execution failed",
					zap.String("command", args.Command),
					zap.Error(err),
					zap.String("stderr", stderr.String()))
				resp := BashResponse{
					Stdout: out.String(),
					Stderr: stderr.String(),
					Status: "failed",
				}
				return t.toJSON(resp)
			}
			resp := BashResponse{
				Stdout: out.String(),
				Stderr: stderr.String(),
				Status: "completed",
			}
			t.logger.Info("Bash command executed successfully",
				zap.String("command", args.Command))
			return t.toJSON(resp)
		case <-timer.C:
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
			t.logger.Warn("Command timed out",
				zap.String("command", args.Command),
				zap.Int("timeout", args.Timeout))
			resp := BashResponse{
				Stdout: out.String(),
				Stderr: stderr.String(),
				Status: "timeout",
			}
			return t.toJSON(resp)
		}
	}

	// Run without timeout
	err := cmd.Run()
	resp := BashResponse{
		Stdout: out.String(),
		Stderr: stderr.String(),
	}
	if err != nil {
		resp.Status = "failed"
		t.logger.Error("Bash command execution failed",
			zap.String("command", args.Command),
			zap.Error(err),
			zap.String("stderr", stderr.String()))
	} else {
		resp.Status = "completed"
		t.logger.Info("Bash command executed successfully",
			zap.String("command", args.Command))
	}
	return t.toJSON(resp)
}

func (t *BashTool) checkStatus(pid int) (string, error) {
	if pid == 0 {
		t.logger.Error("PID is required for status check")
		return "", nil
	}
	cmd, exists := t.processes[pid]
	if !exists {
		resp := BashResponse{
			Status: "not found",
		}
		return t.toJSON(resp)
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		delete(t.processes, pid)
		resp := BashResponse{
			PID:    pid,
			Status: "exited",
		}
		t.logger.Info("Background process has exited", zap.Int("pid", pid))
		return t.toJSON(resp)
	}
	resp := BashResponse{
		PID:    pid,
		Status: "running",
	}
	t.logger.Info("Background process status checked", zap.Int("pid", pid))
	return t.toJSON(resp)
}

func (t *BashTool) killProcess(pid int) (string, error) {
	if pid == 0 {
		t.logger.Error("PID is required for kill")
		return "", nil
	}
	cmd, exists := t.processes[pid]
	if !exists {
		resp := BashResponse{
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
	resp := BashResponse{
		PID:    pid,
		Status: "terminated",
	}
	t.logger.Info("Background process terminated", zap.Int("pid", pid))
	return t.toJSON(resp)
}

func (t *BashTool) toJSON(resp BashResponse) (string, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", err
	}
	return string(data), nil
}
