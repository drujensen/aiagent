package repositories_json

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

// TestGetToolForChat_DistinctInstances is AC3(a): two concurrent chat turns
// invoking the same named, config-bearing tool each receive a distinct
// entities.Tool instance - verified by pointer identity, 100 iterations
// under go test -race.
func TestGetToolForChat_DistinctInstances(t *testing.T) {
	storageDir := t.TempDir()
	toolFactory, err := newTestToolFactory(t)
	if err != nil {
		t.Fatalf("failed to create tool factory: %v", err)
	}
	repo, err := NewJSONToolRepository(storageDir, toolFactory, identityConfigResolver{}, testLogger())
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	toolData := &entities.ToolData{
		Name:          "web-search",
		Description:   "test",
		ToolType:      "WebSearch",
		Configuration: map[string]string{"tavily_api_key": "test-key"},
	}
	if err := repo.CreateToolData(context.Background(), toolData); err != nil {
		t.Fatalf("failed to seed tool data: %v", err)
	}

	const iterations = 100
	instances := make([]entities.Tool, iterations)
	var wg sync.WaitGroup
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tool, err := repo.GetToolForChat(context.Background(), "web-search", nil)
			if err != nil {
				t.Errorf("GetToolForChat failed: %v", err)
				return
			}
			instances[i] = tool
		}(i)
	}
	wg.Wait()

	seen := make(map[entities.Tool]bool, iterations)
	for _, inst := range instances {
		if inst == nil {
			t.Fatal("got a nil tool instance")
		}
		if seen[inst] {
			t.Fatal("two concurrent GetToolForChat calls returned the same instance - expected each call to mint its own")
		}
		seen[inst] = true
	}
}

// TestGetToolForChat_ResolvesConfiguration is AC3(b): the minted instance's
// configuration holds the resolved value, not the literal #{VAR}# placeholder
// - checked via Configuration(), which is exactly what Execute reads from.
func TestGetToolForChat_ResolvesConfiguration(t *testing.T) {
	t.Setenv("TEST_TAVILY_KEY", "real-resolved-value")

	storageDir := t.TempDir()
	toolFactory, err := newTestToolFactory(t)
	if err != nil {
		t.Fatalf("failed to create tool factory: %v", err)
	}
	repo, err := NewJSONToolRepository(storageDir, toolFactory, &envConfigResolver{}, testLogger())
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	toolData := &entities.ToolData{
		Name:          "web-search",
		Description:   "test",
		ToolType:      "WebSearch",
		Configuration: map[string]string{"tavily_api_key": "#{TEST_TAVILY_KEY}#"},
	}
	if err := repo.CreateToolData(context.Background(), toolData); err != nil {
		t.Fatalf("failed to seed tool data: %v", err)
	}

	tool, err := repo.GetToolForChat(context.Background(), "web-search", nil)
	if err != nil {
		t.Fatalf("GetToolForChat failed: %v", err)
	}

	got := tool.Configuration()["tavily_api_key"]
	if got == "#{TEST_TAVILY_KEY}#" {
		t.Fatal("tool configuration still holds the unresolved placeholder - expected it resolved at mint time")
	}
	if got != "real-resolved-value" {
		t.Fatalf("expected resolved config value 'real-resolved-value', got %q", got)
	}
}

// TestGetToolForChat_StatefulToolConfigResolvedAtConstruction is AC3(d): a
// carved-out (Stateful), config-bearing tool holds resolved configuration -
// resolved once at singleton-construction time in reloadToolInstances, since
// it never goes through GetToolForChat's per-turn resolution path.
func TestGetToolForChat_StatefulToolConfigResolvedAtConstruction(t *testing.T) {
	t.Setenv("TEST_WORKSPACE_VAR", "/resolved/workspace")

	storageDir := t.TempDir()
	toolFactory, err := newTestToolFactory(t)
	if err != nil {
		t.Fatalf("failed to create tool factory: %v", err)
	}
	repo, err := NewJSONToolRepository(storageDir, toolFactory, &envConfigResolver{}, testLogger())
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	toolData := &entities.ToolData{
		Name:          "bash-1",
		Description:   "test",
		ToolType:      "Bash", // Stateful: true (ProcessTool)
		Configuration: map[string]string{"workspace": "#{TEST_WORKSPACE_VAR}#"},
	}
	if err := repo.CreateToolData(context.Background(), toolData); err != nil {
		t.Fatalf("failed to seed tool data: %v", err)
	}

	// GetToolForChat on a Stateful tool must return the existing singleton,
	// already resolved - not mint a fresh instance.
	toolA, err := repo.GetToolForChat(context.Background(), "bash-1", nil)
	if err != nil {
		t.Fatalf("GetToolForChat failed: %v", err)
	}
	toolB, err := repo.GetToolForChat(context.Background(), "bash-1", nil)
	if err != nil {
		t.Fatalf("GetToolForChat failed: %v", err)
	}
	if toolA != toolB {
		t.Fatal("expected a Stateful tool to return the same singleton instance across calls, got two different instances")
	}

	got := toolA.Configuration()["workspace"]
	if got == "#{TEST_WORKSPACE_VAR}#" {
		t.Fatal("stateful tool configuration still holds the unresolved placeholder - expected it resolved at construction time")
	}
	if got != "/resolved/workspace" {
		t.Fatalf("expected resolved workspace '/resolved/workspace', got %q", got)
	}
}

// TestProcessTool_ConcurrentExecute is AC3(c) for the carved-out ProcessTool:
// concurrent Execute calls against the shared singleton must not race on the
// processes map (verified by go test -race) and must all succeed.
func TestProcessTool_ConcurrentExecute(t *testing.T) {
	storageDir := t.TempDir()
	toolFactory, err := newTestToolFactory(t)
	if err != nil {
		t.Fatalf("failed to create tool factory: %v", err)
	}
	repo, err := NewJSONToolRepository(storageDir, toolFactory, identityConfigResolver{}, testLogger())
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	toolData := &entities.ToolData{
		Name:          "bash-1",
		Description:   "test",
		ToolType:      "Bash",
		Configuration: map[string]string{"workspace": os.TempDir()},
	}
	if err := repo.CreateToolData(context.Background(), toolData); err != nil {
		t.Fatalf("failed to seed tool data: %v", err)
	}

	tool, err := repo.GetToolForChat(context.Background(), "bash-1", nil)
	if err != nil {
		t.Fatalf("GetToolForChat failed: %v", err)
	}

	const concurrent = 20
	var wg sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tool.Execute(context.Background(), `{"command": "echo hi", "shell": true}`)
			if err != nil {
				t.Errorf("Execute failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

// TestBrowserTool_ConcurrentExecute is AC3(c) for the carved-out BrowserTool:
// concurrent Execute calls against the shared singleton must not race on the
// browser/page fields (verified by go test -race). Skips gracefully if no
// browser binary is available in the environment - this test verifies
// synchronization, not headless-browser availability.
func TestBrowserTool_ConcurrentExecute(t *testing.T) {
	storageDir := t.TempDir()
	toolFactory, err := newTestToolFactory(t)
	if err != nil {
		t.Fatalf("failed to create tool factory: %v", err)
	}
	repo, err := NewJSONToolRepository(storageDir, toolFactory, identityConfigResolver{}, testLogger())
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	toolData := &entities.ToolData{
		Name:          "browser-1",
		Description:   "test",
		ToolType:      "Browser",
		Configuration: map[string]string{"headless": "true"},
	}
	if err := repo.CreateToolData(context.Background(), toolData); err != nil {
		t.Fatalf("failed to seed tool data: %v", err)
	}

	tool, err := repo.GetToolForChat(context.Background(), "browser-1", nil)
	if err != nil {
		t.Fatalf("GetToolForChat failed: %v", err)
	}

	// Probe once, sequentially, so a "no browser available" environment
	// skips cleanly instead of every goroutine failing independently.
	if _, err := tool.Execute(context.Background(), `{"operation": "open", "url": "about:blank"}`); err != nil {
		t.Skipf("skipping: no usable browser in this environment: %v", err)
	}

	const concurrent = 10
	var wg sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tool.Execute(context.Background(), `{"operation": "open", "url": "about:blank"}`)
			if err != nil {
				t.Errorf("Execute failed: %v", err)
			}
		}()
	}
	wg.Wait()
}
