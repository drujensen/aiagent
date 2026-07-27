package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// TestProcessTool_DenyList_RejectsMergeAndForcePush is AC7's negative test:
// it asserts ProcessTool.Execute rejects each of the 5 deny-listed command
// shapes with an error response, rather than checking something trivially
// true for any freshly-created PR (e.g. gh pr view's state).
func TestProcessTool_DenyList_RejectsMergeAndForcePush(t *testing.T) {
	logger := zap.NewNop()
	tool := NewProcessTool("Bash", "test", map[string]string{"workspace": t.TempDir()}, logger)

	denyListed := []string{
		"gh pr merge 42",
		"gh pr merge --auto 42",
		"git push --force origin main",
		"git push -f origin main",
		"git push --force-with-lease origin main",
	}

	for _, cmd := range denyListed {
		t.Run(cmd, func(t *testing.T) {
			argsJSON, err := json.Marshal(map[string]any{"command": cmd, "description": "test"})
			if err != nil {
				t.Fatalf("failed to marshal args: %v", err)
			}

			result, err := tool.Execute(context.Background(), string(argsJSON))
			if err != nil {
				t.Fatalf("Execute returned an error instead of a rejected response: %v", err)
			}

			var resp struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			}
			if err := json.Unmarshal([]byte(result), &resp); err != nil {
				t.Fatalf("failed to unmarshal result %q: %v", result, err)
			}
			if resp.Status != "rejected" {
				t.Errorf("command %q: expected status \"rejected\", got %q (full result: %s)", cmd, resp.Status, result)
			}
			if resp.Error == "" {
				t.Errorf("command %q: expected a non-empty rejection error", cmd)
			}
		})
	}
}

// TestProcessTool_DenyList_AllowsOrdinaryGitPush confirms the deny-list
// doesn't overreach: a plain "git push" with no force flag must reach the
// real command execution path (status "failed" here only because there is
// no git repo/remote in the temp workspace - never "rejected").
func TestProcessTool_DenyList_AllowsOrdinaryGitPush(t *testing.T) {
	logger := zap.NewNop()
	tool := NewProcessTool("Bash", "test", map[string]string{"workspace": t.TempDir()}, logger)

	argsJSON, err := json.Marshal(map[string]any{"command": "git push origin main", "description": "test"})
	if err != nil {
		t.Fatalf("failed to marshal args: %v", err)
	}

	result, err := tool.Execute(context.Background(), string(argsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("failed to unmarshal result %q: %v", result, err)
	}
	if resp.Status == "rejected" {
		t.Errorf("ordinary \"git push\" should never be rejected by the deny-list, got status %q", resp.Status)
	}
}

// TestGitPRTool_NoMergeAutoSquashConstruction is AC7's repo-scoped grep
// check, run as a Go test so it's a real deliverable rather than a manual
// step. It has an explicit precondition that git_pr_tool.go exists, so a
// grep against a not-yet-existing file can't trivially "pass" with zero
// matches. It scans every non-test .go file in this package for a quoted
// CLI argument literal that would construct a merge, auto-merge, or squash
// operation.
func TestGitPRTool_NoMergeAutoSquashConstruction(t *testing.T) {
	const targetFile = "git_pr_tool.go"
	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("precondition failed: %s must exist for this check to be meaningful: %v", targetFile, err)
	}

	forbidden := []string{`"merge"`, `"--auto"`, `"--squash"`}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read internal/impl/tools/: %v", err)
	}

	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("failed to read %s: %v", name, err)
		}
		content := string(data)
		for _, f := range forbidden {
			if strings.Contains(content, f) {
				violations = append(violations, name+" contains forbidden literal "+f)
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("found merge/auto/squash argument construction:\n%s", strings.Join(violations, "\n"))
	}
}
