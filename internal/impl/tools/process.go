package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

type ProcessInfo struct {
	Cmd          *exec.Cmd
	Stdin        io.WriteCloser
	Stdout       io.ReadCloser
	Stderr       io.ReadCloser
	StdoutBuffer *bytes.Buffer
	StderrBuffer *bytes.Buffer
}

type ProcessTool struct {
	name          string
	description   string
	configuration map[string]string // Includes "workspace"
	logger        *zap.Logger
	processes     map[int]*ProcessInfo // Track background processes by PID
}

func NewProcessTool(name, description string, configuration map[string]string, logger *zap.Logger) *ProcessTool {
	return &ProcessTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		processes:     make(map[int]*ProcessInfo),
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
	return fmt.Sprintf("%s\n\nParameters:\n- command: shell command to execute\n- shell: run in shell (optional, default false)\n- timeout: timeout in seconds (optional)\n\nNote: Output limited to 4096 tokens.", t.Description())
}

func (t *ProcessTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "command",
			Type:        "string",
			Description: "The full command to execute (e.g., 'git clone https://github.com/repo')",
			Required:    true,
		},
		{
			Name:        "shell",
			Type:        "boolean",
			Description: "Run the command through the OS default shell to inherit PATH and environment",
			Required:    false,
		},
		{
			Name:        "input",
			Type:        "string",
			Description: "Input data to send to the process's stdin",
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
			Enum:        []string{"run", "write", "read", "status", "kill"},
			Description: "Action to perform: run (default), write (send to stdin), read (read from stdout/stderr), status (check PID), or kill (stop PID)",
			Required:    false,
		},
	}
}

type ProcessResponse struct {
	Command string `json:"command"`
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
	PID     int    `json:"pid,omitempty"`
	Status  string `json:"status,omitempty"`
}

type ProcessArgs struct {
	Command    string   `json:"command"`
	Shell      bool     `json:"shell"`
	Input      string   `json:"input"`
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
		results, err := t.runCommand(args, workspace)
		if err != nil {
			return results, err
		}
		return t.formatRunOutput(results)
	case "write":
		return t.writeToProcess(args)
	case "read":
		return t.readFromProcess(args)
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

func (t *ProcessTool) runCommand(args ProcessArgs, workspace string) (string, error) {
	// Parse full command if not shell mode
	var cmd *exec.Cmd
	cmdArgs := splitShellArgs(args.Command)
	if args.Shell {
		var shell, flag string
		if runtime.GOOS == "windows" {
			shell = "pwsh"
			flag = "-Command"
		} else {
			shell = "bash"
			flag = "-c"
		}
		cmd = exec.Command(shell, flag, args.Command)
	} else {
		if len(cmdArgs) == 0 {
			return "", fmt.Errorf("no command specified")
		}
		cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
	}
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), args.Env...)

	if args.Background {
		cmd.Stdout = nil
		cmd.Stderr = nil
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", err
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return "", err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return "", err
		}
		err = cmd.Start()
		if err != nil {
			t.logger.Error("Failed to start background command",
				zap.String("command", args.Command),
				zap.Strings("arguments", cmdArgs),
				zap.Error(err))
			return "", err
		}
		pid := cmd.Process.Pid
		pi := &ProcessInfo{
			Cmd:          cmd,
			Stdin:        stdin,
			Stdout:       stdout,
			Stderr:       stderr,
			StdoutBuffer: &bytes.Buffer{},
			StderrBuffer: &bytes.Buffer{},
		}
		go io.Copy(pi.StdoutBuffer, stdout)
		go io.Copy(pi.StderrBuffer, stderr)
		t.processes[pid] = pi
		t.logger.Info("Background command started",
			zap.String("command", args.Command),
			zap.Strings("arguments", cmdArgs),
			zap.Int("pid", pid))
		if args.Input != "" {
			_, err = io.WriteString(stdin, args.Input+"\n")
			if err != nil {
				t.logger.Warn("Failed to write initial input", zap.Error(err))
			}
		}
		resp := ProcessResponse{
			Command: args.Command,
			Stdout:  "Command started in background",
			PID:     pid,
			Status:  "running",
		}
		return t.toJSON(resp)
	} else {
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if args.Input != "" {
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return "", err
			}
			go func() {
				defer stdin.Close()
				io.WriteString(stdin, args.Input+"\n")
			}()
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
						zap.String("command", args.Command),
						zap.Strings("arguments", cmdArgs),
						zap.Error(err),
						zap.String("stdout", out.String()),
						zap.String("stderr", stderr.String()))
					resp := ProcessResponse{
						Command: args.Command,
						Stdout:  out.String(),
						Stderr:  stderr.String(),
						Status:  "failed",
					}
					return t.toJSON(resp)
				}
				resp := ProcessResponse{
					Command: args.Command,
					Stdout:  out.String(),
					Stderr:  stderr.String(),
					Status:  "completed",
				}
				t.logger.Info("Command executed successfully",
					zap.String("command", args.Command),
					zap.Strings("arguments", cmdArgs))
				return t.toJSON(resp)
			case <-timer.C:
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
				t.logger.Warn("Command timed out",
					zap.String("command", args.Command),
					zap.Strings("arguments", cmdArgs),
					zap.Int("timeout", args.Timeout))
				resp := ProcessResponse{
					Command: args.Command,
					Stdout:  out.String(),
					Stderr:  stderr.String(),
					Status:  "timeout",
				}
				return t.toJSON(resp)
			}
		}

		err := cmd.Run()
		resp := ProcessResponse{
			Command: args.Command,
			Stdout:  out.String(),
			Stderr:  stderr.String(),
		}
		if err != nil {
			resp.Status = "failed"
			t.logger.Error("Command execution failed",
				zap.String("command", args.Command),
				zap.Strings("arguments", cmdArgs),
				zap.Error(err),
				zap.String("stderr", stderr.String()))
		} else {
			resp.Status = "completed"
			t.logger.Info("Command executed successfully",
				zap.String("command", args.Command),
				zap.Strings("arguments", cmdArgs))
		}
		return t.toJSON(resp)
	}
}

func (t *ProcessTool) checkStatus(pid int) (string, error) {
	if pid == 0 {
		t.logger.Error("PID is required for status check")
		return "", fmt.Errorf("PID is required for status check")
	}
	pi, exists := t.processes[pid]
	if !exists {
		resp := ProcessResponse{
			Command: "status",
			Status:  "not found",
		}
		return t.toJSON(resp)
	}
	if pi.Cmd.ProcessState != nil && pi.Cmd.ProcessState.Exited() {
		pi.Stdin.Close()
		pi.Stdout.Close()
		pi.Stderr.Close()
		delete(t.processes, pid)
		resp := ProcessResponse{
			Command: "status",
			PID:     pid,
			Status:  "exited",
		}
		t.logger.Info("Background process has exited", zap.Int("pid", pid))
		return t.toJSON(resp)
	}
	resp := ProcessResponse{
		Command: "status",
		PID:     pid,
		Status:  "running",
	}
	t.logger.Info("Background process status checked", zap.Int("pid", pid))
	return t.toJSON(resp)
}

func (t *ProcessTool) killProcess(pid int) (string, error) {
	if pid == 0 {
		t.logger.Error("PID is required for kill")
		return "", fmt.Errorf("PID is required for kill")
	}
	pi, exists := t.processes[pid]
	if !exists {
		resp := ProcessResponse{
			Command: "kill",
			Status:  "not found",
		}
		return t.toJSON(resp)
	}
	err := pi.Cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.logger.Error("Failed to terminate process",
			zap.Int("pid", pid),
			zap.Error(err))
		return "", err
	}
	pi.Stdin.Close()
	delete(t.processes, pid)
	resp := ProcessResponse{
		Command: "kill",
		PID:     pid,
		Status:  "terminated",
	}
	t.logger.Info("Background process terminated", zap.Int("pid", pid))
	return t.toJSON(resp)
}

func (t *ProcessTool) formatRunOutput(jsonOutput string) (string, error) {
	// Parse the JSON response
	var resp ProcessResponse
	if err := json.Unmarshal([]byte(jsonOutput), &resp); err != nil {
		// If parsing fails, return original output
		return jsonOutput, nil
	}

	// Create TUI-friendly summary
	var summary strings.Builder

	// Command and status
	summary.WriteString(fmt.Sprintf("⚙️  %s\n", resp.Command))
	summary.WriteString(fmt.Sprintf("📊 Status: %s\n", resp.Status))

	if resp.PID > 0 {
		summary.WriteString(fmt.Sprintf("🆔 PID: %d\n", resp.PID))
	}

	summary.WriteString("\n")

	// Handle stdout
	if resp.Stdout != "" {
		lines := strings.Split(strings.TrimSpace(resp.Stdout), "\n")
		totalLines := len(lines)

		summary.WriteString(fmt.Sprintf("📤 Stdout (%d lines):\n", totalLines))

		// Show first 10 lines
		previewLines := 10
		if totalLines < previewLines {
			previewLines = totalLines
		}

		for i := 0; i < previewLines; i++ {
			summary.WriteString(fmt.Sprintf("   %s\n", lines[i]))
		}

		if totalLines > 10 {
			summary.WriteString(fmt.Sprintf("   ... and %d more lines\n", totalLines-10))
		}
	}

	// Handle stderr
	if resp.Stderr != "" {
		if resp.Stdout != "" {
			summary.WriteString("\n")
		}

		lines := strings.Split(strings.TrimSpace(resp.Stderr), "\n")
		totalLines := len(lines)

		summary.WriteString(fmt.Sprintf("⚠️  Stderr (%d lines):\n", totalLines))

		// Show first 5 lines of stderr
		previewLines := 5
		if totalLines < previewLines {
			previewLines = totalLines
		}

		for i := 0; i < previewLines; i++ {
			summary.WriteString(fmt.Sprintf("   %s\n", lines[i]))
		}

		if totalLines > 5 {
			summary.WriteString(fmt.Sprintf("   ... and %d more lines\n", totalLines-5))
		}

		// Add guidance for failed commands
		if resp.Status == "failed" {
			summary.WriteString("\n💡 Command failed. Common next steps:\n")
			summary.WriteString("   - Analyze the error messages above\n")
			summary.WriteString("   - Check command syntax and arguments\n")
			summary.WriteString("   - Verify file paths and permissions exist\n")
			summary.WriteString("   - Fix any issues in source code or configuration\n")
			summary.WriteString("   - Try running the command again after addressing errors\n")
		}
	}

	// Create JSON response with summary for TUI and full data for AI
	response := struct {
		Summary     string `json:"summary"`
		Command     string `json:"command"`
		Stdout      string `json:"stdout"`
		Stderr      string `json:"stderr"`
		PID         int    `json:"pid,omitempty"`
		Status      string `json:"status"`
		Recoverable bool   `json:"recoverable,omitempty"` // Indicates if failed commands can be retried after fixing
	}{
		Summary:     summary.String(),
		Command:     resp.Command,
		Stdout:      resp.Stdout,
		Stderr:      resp.Stderr,
		PID:         resp.PID,
		Status:      resp.Status,
		Recoverable: resp.Status == "failed", // Failed commands are typically recoverable by fixing issues
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal process response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	return string(jsonResult), nil
}

func (t *ProcessTool) formatReadOutput(jsonOutput string) (string, error) {
	// Parse the JSON response
	var resp ProcessResponse
	if err := json.Unmarshal([]byte(jsonOutput), &resp); err != nil {
		// If parsing fails, return original output
		return jsonOutput, nil
	}

	// Create TUI-friendly summary
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("📖 Reading from PID %d\n", resp.PID))

	// Handle stdout
	if resp.Stdout != "" {
		lines := strings.Split(strings.TrimSpace(resp.Stdout), "\n")
		totalLines := len(lines)

		summary.WriteString(fmt.Sprintf("📤 Stdout (%d lines):\n", totalLines))

		// Show first 10 lines
		previewLines := 10
		if totalLines < previewLines {
			previewLines = totalLines
		}

		for i := 0; i < previewLines; i++ {
			summary.WriteString(fmt.Sprintf("   %s\n", lines[i]))
		}

		if totalLines > 10 {
			summary.WriteString(fmt.Sprintf("   ... and %d more lines\n", totalLines-10))
		}
	}

	// Handle stderr
	if resp.Stderr != "" {
		if resp.Stdout != "" {
			summary.WriteString("\n")
		}

		lines := strings.Split(strings.TrimSpace(resp.Stderr), "\n")
		totalLines := len(lines)

		summary.WriteString(fmt.Sprintf("⚠️  Stderr (%d lines):\n", totalLines))

		// Show first 5 lines of stderr
		previewLines := 5
		if totalLines < previewLines {
			previewLines = totalLines
		}

		for i := 0; i < previewLines; i++ {
			summary.WriteString(fmt.Sprintf("   %s\n", lines[i]))
		}

		if totalLines > 5 {
			summary.WriteString(fmt.Sprintf("   ... and %d more lines\n", totalLines-5))
		}
	}

	// Create JSON response with summary for TUI and full data for AI
	response := struct {
		Summary string `json:"summary"`
		Command string `json:"command"`
		Stdout  string `json:"stdout"`
		Stderr  string `json:"stderr"`
		PID     int    `json:"pid,omitempty"`
		Status  string `json:"status"`
	}{
		Summary: summary.String(),
		Command: resp.Command,
		Stdout:  resp.Stdout,
		Stderr:  resp.Stderr,
		PID:     resp.PID,
		Status:  resp.Status,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal process read response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	return string(jsonResult), nil
}

func (t *ProcessTool) toJSON(resp ProcessResponse) (string, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", err
	}
	return string(data), nil
}

func (t *ProcessTool) writeToProcess(args ProcessArgs) (string, error) {
	if args.PID == 0 {
		return "", fmt.Errorf("PID required for write")
	}
	pi, exists := t.processes[args.PID]
	if !exists {
		return "", fmt.Errorf("process not found")
	}
	if args.Input == "" {
		return "", fmt.Errorf("input required for write")
	}
	_, err := io.WriteString(pi.Stdin, args.Input+"\n")
	if err != nil {
		return "", err
	}
	resp := ProcessResponse{
		Command: "write",
		Status:  "written",
	}
	return t.toJSON(resp)
}

func (t *ProcessTool) readFromProcess(args ProcessArgs) (string, error) {
	if args.PID == 0 {
		return "", fmt.Errorf("PID required for read")
	}
	pi, exists := t.processes[args.PID]
	if !exists {
		return "", fmt.Errorf("process not found")
	}
	stdout := pi.StdoutBuffer.String()
	stderr := pi.StderrBuffer.String()
	pi.StdoutBuffer.Reset()
	pi.StderrBuffer.Reset()
	resp := ProcessResponse{
		Command: "read",
		Stdout:  stdout,
		Stderr:  stderr,
		Status:  "read",
	}
	jsonOutput, err := t.toJSON(resp)
	if err != nil {
		return "", err
	}
	return t.formatReadOutput(jsonOutput)
}

var _ entities.Tool = (*ProcessTool)(nil)
