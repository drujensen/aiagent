package tools

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// initGitRepo creates a real git repository in t.TempDir() with an initial
// commit on its default branch, so tests can exercise GitPRTool's actual
// "branch"/"commit" command execution path, not just argument validation.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}
	run("git", "add", "README.md")
	run("git", "commit", "-m", "initial commit")
	return dir
}

func TestGitPRTool_RejectsDisallowedBranchPrefix(t *testing.T) {
	logger := zap.NewNop()
	tool := NewGitPRTool("GitPR", "test", map[string]string{"workspace": t.TempDir()}, logger)

	for _, action := range []string{"branch", "push"} {
		t.Run(action, func(t *testing.T) {
			argsJSON, err := json.Marshal(map[string]any{"action": action, "branch_name": "main"})
			if err != nil {
				t.Fatalf("failed to marshal args: %v", err)
			}
			_, err = tool.Execute(context.Background(), string(argsJSON))
			if err == nil {
				t.Fatalf("expected an error for a branch_name not prefixed with %q", allowedBranchPrefix)
			}
			if !strings.Contains(err.Error(), allowedBranchPrefix) {
				t.Errorf("expected error to mention required prefix %q, got: %v", allowedBranchPrefix, err)
			}
		})
	}
}

func TestGitPRTool_AllowsPrefixedBranchName(t *testing.T) {
	if err := requireAllowedBranch("aiagent/auto/my-feature"); err != nil {
		t.Errorf("expected a correctly-prefixed branch name to pass, got: %v", err)
	}
}

func TestGitPRTool_UnsupportedAction(t *testing.T) {
	logger := zap.NewNop()
	tool := NewGitPRTool("GitPR", "test", map[string]string{"workspace": t.TempDir()}, logger)

	argsJSON, err := json.Marshal(map[string]any{"action": "merge"})
	if err != nil {
		t.Fatalf("failed to marshal args: %v", err)
	}
	_, err = tool.Execute(context.Background(), string(argsJSON))
	if err == nil {
		t.Fatal("expected an error for an unsupported action")
	}
	if !strings.Contains(err.Error(), "unsupported action") {
		t.Errorf("expected \"unsupported action\" error, got: %v", err)
	}
}

func TestGitPRTool_CommitRequiresMessageAndFiles(t *testing.T) {
	logger := zap.NewNop()
	tool := NewGitPRTool("GitPR", "test", map[string]string{"workspace": t.TempDir()}, logger)

	cases := []struct {
		name string
		args map[string]any
	}{
		{"missing message", map[string]any{"action": "commit", "files": []string{"a.go"}}},
		{"missing files", map[string]any{"action": "commit", "message": "msg"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsJSON, err := json.Marshal(tc.args)
			if err != nil {
				t.Fatalf("failed to marshal args: %v", err)
			}
			_, err = tool.Execute(context.Background(), string(argsJSON))
			if err == nil {
				t.Fatalf("expected an error for %s", tc.name)
			}
		})
	}
}

// TestGitPRTool_BranchAndCommit_Succeeds exercises the real "branch" and
// "commit" command execution against an actual git repository, proving the
// tool's argv construction genuinely works end-to-end rather than only
// being validated by pre-execution argument checks.
func TestGitPRTool_BranchAndCommit_Succeeds(t *testing.T) {
	repoDir := initGitRepo(t)
	logger := zap.NewNop()
	tool := NewGitPRTool("GitPR", "test", map[string]string{"workspace": repoDir}, logger)
	ctx := context.Background()

	branchArgs, err := json.Marshal(map[string]any{"action": "branch", "branch_name": "aiagent/auto/phase5-test"})
	if err != nil {
		t.Fatalf("failed to marshal args: %v", err)
	}
	if _, err := tool.Execute(ctx, string(branchArgs)); err != nil {
		t.Fatalf("branch action failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoDir, "new-file.txt"), []byte("content\n"), 0644); err != nil {
		t.Fatalf("failed to write new-file.txt: %v", err)
	}

	commitArgs, err := json.Marshal(map[string]any{
		"action":  "commit",
		"files":   []string{"new-file.txt"},
		"message": "add new-file.txt",
	})
	if err != nil {
		t.Fatalf("failed to marshal args: %v", err)
	}
	if _, err := tool.Execute(ctx, string(commitArgs)); err != nil {
		t.Fatalf("commit action failed: %v", err)
	}

	logCmd := exec.Command("git", "log", "--oneline", "-1")
	logCmd.Dir = repoDir
	out, err := logCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	if !strings.Contains(string(out), "add new-file.txt") {
		t.Errorf("expected the new commit to be on HEAD, git log showed: %s", out)
	}

	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = repoDir
	branchOut, err := branchCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git branch --show-current failed: %v", err)
	}
	if strings.TrimSpace(string(branchOut)) != "aiagent/auto/phase5-test" {
		t.Errorf("expected to be on branch aiagent/auto/phase5-test, got %q", strings.TrimSpace(string(branchOut)))
	}
}

func TestGitPRTool_PRCreateRequiresTitleAndBody(t *testing.T) {
	logger := zap.NewNop()
	tool := NewGitPRTool("GitPR", "test", map[string]string{"workspace": t.TempDir()}, logger)

	cases := []struct {
		name string
		args map[string]any
	}{
		{"missing title", map[string]any{"action": "pr_create", "body": "body"}},
		{"missing body", map[string]any{"action": "pr_create", "title": "title"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			argsJSON, err := json.Marshal(tc.args)
			if err != nil {
				t.Fatalf("failed to marshal args: %v", err)
			}
			_, err = tool.Execute(context.Background(), string(argsJSON))
			if err == nil {
				t.Fatalf("expected an error for %s", tc.name)
			}
		})
	}
}
