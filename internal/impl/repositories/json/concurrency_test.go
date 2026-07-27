package repositories_json

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/impl/tools"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func newTestToolFactory(t *testing.T) (*tools.ToolFactory, error) {
	t.Helper()
	return tools.NewToolFactory()
}

const concurrencyGoroutines = 50

// TestJsonChatRepository_ConcurrentAccess exercises AC4(a): 50 goroutines
// each creating a distinct Chat concurrently must not panic under -race,
// must not lose any entity, and must leave a parseable JSON file behind.
func TestJsonChatRepository_ConcurrentAccess(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONChatRepository(storageDir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrencyGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			chat := entities.NewChat("agent-1", "model-1", "chat")
			chat.ID = ""
			if err := repo.CreateChat(context.Background(), chat); err != nil {
				t.Errorf("CreateChat failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	chats, err := repo.ListChats(context.Background())
	if err != nil {
		t.Fatalf("ListChats failed: %v", err)
	}
	if len(chats) != concurrencyGoroutines {
		t.Fatalf("expected %d chats, got %d (lost update)", concurrencyGoroutines, len(chats))
	}

	assertJSONParses(t, filepath.Join(storageDir, "chats.json"))
}

func TestJsonAgentRepository_ConcurrentAccess(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONAgentRepository(storageDir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrencyGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			agent := &entities.Agent{Name: "agent", Tools: []string{"Bash"}}
			if err := repo.CreateAgent(context.Background(), agent); err != nil {
				t.Errorf("CreateAgent failed: %v", err)
			}
		}()
	}
	wg.Wait()

	agents, err := repo.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != concurrencyGoroutines {
		t.Fatalf("expected %d agents, got %d (lost update)", concurrencyGoroutines, len(agents))
	}

	assertJSONParses(t, filepath.Join(storageDir, "agents.json"))
}

func TestJsonModelRepository_ConcurrentAccess(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONModelRepository(storageDir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrencyGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			model := &entities.Model{Name: "model", ProviderID: "provider-1", ModelName: "test-model"}
			if err := repo.CreateModel(context.Background(), model); err != nil {
				t.Errorf("CreateModel failed: %v", err)
			}
		}()
	}
	wg.Wait()

	models, err := repo.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}
	if len(models) != concurrencyGoroutines {
		t.Fatalf("expected %d models, got %d (lost update)", concurrencyGoroutines, len(models))
	}

	assertJSONParses(t, filepath.Join(storageDir, "models.json"))
}

func TestJsonProviderRepository_ConcurrentAccess(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONProviderRepository(storageDir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrencyGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			provider := &entities.Provider{Name: "provider", Type: "generic", Models: []entities.ModelPricing{}}
			if err := repo.CreateProvider(context.Background(), provider); err != nil {
				t.Errorf("CreateProvider failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	providers, err := repo.ListProviders(context.Background())
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != concurrencyGoroutines {
		t.Fatalf("expected %d providers, got %d (lost update)", concurrencyGoroutines, len(providers))
	}

	assertJSONParses(t, filepath.Join(storageDir, "providers.json"))
}

func TestJsonTaskRepository_ConcurrentAccess(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONTaskRepository(storageDir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrencyGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task := entities.NewTask("task", "content", entities.TaskPriorityMedium)
			task.ID = ""
			if err := repo.CreateTask(context.Background(), task); err != nil {
				t.Errorf("CreateTask failed: %v", err)
			}
		}()
	}
	wg.Wait()

	tasks, err := repo.ListTasks(context.Background())
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != concurrencyGoroutines {
		t.Fatalf("expected %d tasks, got %d (lost update)", concurrencyGoroutines, len(tasks))
	}

	assertJSONParses(t, filepath.Join(storageDir, "tasks.json"))
}

func TestJsonToolRepository_ConcurrentAccess(t *testing.T) {
	storageDir := t.TempDir()
	toolFactory, err := newTestToolFactory(t)
	if err != nil {
		t.Fatalf("failed to create tool factory: %v", err)
	}
	repo, err := NewJSONToolRepository(storageDir, toolFactory, testLogger())
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrencyGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			toolData := &entities.ToolData{
				Name:          uniqueName("bash-tool", n),
				Description:   "test tool",
				ToolType:      "Bash",
				Configuration: map[string]string{"workspace": "/tmp"},
			}
			if err := repo.CreateToolData(context.Background(), toolData); err != nil {
				t.Errorf("CreateToolData failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	toolDataList, err := repo.ListToolData(context.Background())
	if err != nil {
		t.Fatalf("ListToolData failed: %v", err)
	}
	if len(toolDataList) != concurrencyGoroutines {
		t.Fatalf("expected %d tool data records, got %d (lost update)", concurrencyGoroutines, len(toolDataList))
	}

	assertJSONParses(t, filepath.Join(storageDir, "tools.json"))
}

// TestJsonChatRepository_ServiceLayerConcurrency is AC4(b): it reproduces the
// exact SendMessage-shaped sequence chat_service.go performs (GetChat ->
// mutate Messages -> UpdateChat -> UpdateUsage -> UpdateChat again)
// concurrently with a reader that dereferences the nested Usage/ToolCalls
// fields on every returned copy - not just calling ListChats/GetChat, since
// -race can only detect an actual conflicting access, not merely a shared
// pointer that's never dereferenced. This is the test that would have caught
// the Chat.Usage/Message.Usage/ToolCalls aliasing bug found during review.
func TestJsonChatRepository_ServiceLayerConcurrency(t *testing.T) {
	storageDir := t.TempDir()
	repo, err := NewJSONChatRepository(storageDir)
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	seed := entities.NewChat("agent-1", "model-1", "chat")
	if err := repo.CreateChat(context.Background(), seed); err != nil {
		t.Fatalf("failed to seed chat: %v", err)
	}
	chatID := seed.ID

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer: repeatedly performs the exact sequence chat_service.SendMessage
	// performs - load, append a message, save, recompute usage in place,
	// save again.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			chat, err := repo.GetChat(context.Background(), chatID)
			if err != nil {
				t.Errorf("GetChat failed: %v", err)
				return
			}
			msg := entities.NewMessage("assistant", "reply")
			msg.AddUsage(10, 20, 1.0, 2.0)
			msg.ToolCalls = append(msg.ToolCalls, entities.ToolCall{ID: "tc-1", Type: "function"})
			chat.Messages = append(chat.Messages, *msg)
			if err := repo.UpdateChat(context.Background(), chat); err != nil {
				t.Errorf("UpdateChat failed: %v", err)
				return
			}
			chat.UpdateUsage()
			if err := repo.UpdateChat(context.Background(), chat); err != nil {
				t.Errorf("UpdateChat (post-usage) failed: %v", err)
				return
			}
		}
		close(stop)
	}()

	// Reader: repeatedly lists and gets the chat, and actually dereferences
	// the nested pointer/slice fields on every returned copy.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}

			chats, err := repo.ListChats(context.Background())
			if err != nil {
				t.Errorf("ListChats failed: %v", err)
				return
			}
			for _, c := range chats {
				if c.Usage != nil {
					_ = c.Usage.TotalTokens // dereference - gives -race a real participant
				}
				for _, m := range c.Messages {
					if m.Usage != nil {
						_ = m.Usage.Cost
					}
					if len(m.ToolCalls) > 0 {
						_ = m.ToolCalls[0].ID
					}
				}
			}

			single, err := repo.GetChat(context.Background(), chatID)
			if err != nil {
				t.Errorf("GetChat failed: %v", err)
				return
			}
			if single.Usage != nil {
				_ = single.Usage.TotalTokens
			}
		}
	}()

	wg.Wait()

	final, err := repo.GetChat(context.Background(), chatID)
	if err != nil {
		t.Fatalf("final GetChat failed: %v", err)
	}
	if len(final.Messages) != 200 {
		t.Fatalf("expected 200 messages after the writer loop, got %d (lost update)", len(final.Messages))
	}

	assertJSONParses(t, filepath.Join(storageDir, "chats.json"))
}

func assertJSONParses(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("%s is not valid JSON after concurrent writes: %v", path, err)
	}
}

func uniqueName(prefix string, n int) string {
	return prefix + "-" + string(rune('a'+n%26)) + string(rune('0'+n/26))
}
