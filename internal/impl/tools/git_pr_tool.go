package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

// allowedBranchPrefix is enforced on every branch this tool creates or
// pushes to, so this tool can never touch a human-owned branch.
const allowedBranchPrefix = "aiagent/auto/"

// GitPRTool provides a narrow, purpose-built surface for the automated
// pipeline to create a branch, commit, push, and open a pull request. It
// deliberately supports only these four actions - there is no merge, auto-merge,
// or squash action anywhere in this file, structurally (not just by runtime
// check) preventing this tool from ever merging a PR.
type GitPRTool struct {
	name          string
	description   string
	configuration map[string]string // Includes "workspace"
	logger        *zap.Logger
}

func NewGitPRTool(name, description string, configuration map[string]string, logger *zap.Logger) *GitPRTool {
	return &GitPRTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *GitPRTool) Name() string { return t.name }

func (t *GitPRTool) Description() string { return t.description }

func (t *GitPRTool) Configuration() map[string]string { return t.configuration }

func (t *GitPRTool) UpdateConfiguration(config map[string]string) { t.configuration = config }

func (t *GitPRTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- action: One of \"branch\", \"commit\", \"push\", \"pr_create\".\n- branch_name: Branch name, required for \"branch\" and \"push\"; must start with %q.\n- files: File paths to stage, required for \"commit\".\n- message: Commit message, required for \"commit\".\n- title: Pull request title, required for \"pr_create\".\n- body: Pull request body, required for \"pr_create\".\n- base: Pull request base branch, optional for \"pr_create\".", t.Description(), allowedBranchPrefix)
}

func (t *GitPRTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"branch", "commit", "push", "pr_create"},
				"description": "The git/PR operation to perform.",
			},
			"branch_name": map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("Branch name; must start with %q.", allowedBranchPrefix),
			},
			"files": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "File paths to stage for commit.",
			},
			"message": map[string]any{
				"type":        "string",
				"description": "Commit message.",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "Pull request title.",
			},
			"body": map[string]any{
				"type":        "string",
				"description": "Pull request body.",
			},
			"base": map[string]any{
				"type":        "string",
				"description": "Pull request base branch.",
			},
		},
		"required":             []string{"action"},
		"additionalProperties": false,
	}
}

type gitPRArgs struct {
	Action     string   `json:"action"`
	BranchName string   `json:"branch_name"`
	Files      []string `json:"files"`
	Message    string   `json:"message"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Base       string   `json:"base"`
}

func (t *GitPRTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args gitPRArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current directory: %w", err)
		}
	}

	switch args.Action {
	case "branch":
		return t.branch(ctx, workspace, args)
	case "commit":
		return t.commit(ctx, workspace, args)
	case "push":
		return t.push(ctx, workspace, args)
	case "pr_create":
		return t.prCreate(ctx, workspace, args)
	default:
		return "", fmt.Errorf("unsupported action: %s", args.Action)
	}
}

func requireAllowedBranch(branchName string) error {
	if !strings.HasPrefix(branchName, allowedBranchPrefix) {
		return fmt.Errorf("branch_name must start with %q, got %q", allowedBranchPrefix, branchName)
	}
	return nil
}

func (t *GitPRTool) branch(ctx context.Context, workspace string, args gitPRArgs) (string, error) {
	if args.BranchName == "" {
		return "", fmt.Errorf("branch_name is required")
	}
	if err := requireAllowedBranch(args.BranchName); err != nil {
		return "", err
	}
	return t.run(ctx, workspace, "git", "checkout", "-b", args.BranchName)
}

func (t *GitPRTool) commit(ctx context.Context, workspace string, args gitPRArgs) (string, error) {
	if args.Message == "" {
		return "", fmt.Errorf("message is required")
	}
	if len(args.Files) == 0 {
		return "", fmt.Errorf("files is required")
	}
	addArgs := append([]string{"add"}, args.Files...)
	if out, err := t.run(ctx, workspace, "git", addArgs...); err != nil {
		return out, err
	}
	return t.run(ctx, workspace, "git", "commit", "-m", args.Message)
}

func (t *GitPRTool) push(ctx context.Context, workspace string, args gitPRArgs) (string, error) {
	if args.BranchName == "" {
		return "", fmt.Errorf("branch_name is required")
	}
	if err := requireAllowedBranch(args.BranchName); err != nil {
		return "", err
	}
	return t.run(ctx, workspace, "git", "push", "-u", "origin", args.BranchName)
}

func (t *GitPRTool) prCreate(ctx context.Context, workspace string, args gitPRArgs) (string, error) {
	if args.Title == "" {
		return "", fmt.Errorf("title is required")
	}
	if args.Body == "" {
		return "", fmt.Errorf("body is required")
	}
	cmdArgs := []string{"pr", "create", "--title", args.Title, "--body", args.Body}
	if args.Base != "" {
		cmdArgs = append(cmdArgs, "--base", args.Base)
	}
	return t.run(ctx, workspace, "gh", cmdArgs...)
}

// run executes name with args as a direct argv (never through a shell), so
// there is no shell metacharacter or injection surface here regardless of
// what a caller passes in title/body/message/file path fields.
func (t *GitPRTool) run(ctx context.Context, workspace, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.logger.Error("git/pr command failed",
			zap.String("command", name),
			zap.Strings("args", args),
			zap.Error(err),
			zap.String("output", string(out)))
		return string(out), fmt.Errorf("%s %s failed: %w", name, strings.Join(args, " "), err)
	}
	t.logger.Info("git/pr command succeeded", zap.String("command", name), zap.Strings("args", args))
	return string(out), nil
}

func (t *GitPRTool) DisplayName(ui string, arguments string) (string, string) {
	var args struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.Action != "" {
		return t.Name(), args.Action
	}
	return t.Name(), ""
}

func (t *GitPRTool) FormatResult(ui string, result string, diff string, arguments string) string {
	return result
}

var _ entities.Tool = (*GitPRTool)(nil)
