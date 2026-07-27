package repositories_json

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestMigrateDuplicateProviders(t *testing.T) {
	storageDir := t.TempDir()

	for _, name := range []string{"providers_dup.json", "models_dup.json", "chats_dup.json"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("failed to read fixture %s: %v", name, err)
		}
		dest := map[string]string{
			"providers_dup.json": "providers.json",
			"models_dup.json":    "models.json",
			"chats_dup.json":     "chats.json",
		}[name]
		if err := os.WriteFile(filepath.Join(storageDir, dest), data, 0644); err != nil {
			t.Fatalf("failed to seed fixture %s: %v", name, err)
		}
	}

	logger := zap.NewNop()

	if err := MigrateDuplicateProviders(storageDir, logger); err != nil {
		t.Fatalf("MigrateDuplicateProviders failed: %v", err)
	}

	providers := readArray(t, filepath.Join(storageDir, "providers.json"))
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers after dedup, got %d", len(providers))
	}
	togetherCount := 0
	for _, p := range providers {
		if p["name"] == "Together AI" {
			togetherCount++
			if p["id"] != "12345678-1234-1234-1234-123456789012" {
				t.Fatalf("expected surviving Together AI provider ID to be the lexicographically smaller UUID, got %v", p["id"])
			}
		}
	}
	if togetherCount != 1 {
		t.Fatalf("expected exactly 1 Together AI provider, got %d", togetherCount)
	}

	models := readArray(t, filepath.Join(storageDir, "models.json"))
	if len(models) != 2 {
		t.Fatalf("expected 2 models after dedup (1 openai + 1 together), got %d", len(models))
	}
	for _, m := range models {
		if m["provider_id"] == "FE6F981E-CA93-46BE-9B8B-0321A47A64E4" {
			t.Fatalf("model still references removed provider ID: %+v", m)
		}
	}

	chats := readArray(t, filepath.Join(storageDir, "chats.json"))
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}
	if chats[0]["model_id"] != "bbbbbbbb-0000-0000-0000-000000000001" {
		t.Fatalf("expected chat to be re-pointed to the surviving model ID, got %v", chats[0]["model_id"])
	}

	for _, f := range []string{"providers.json.bak", "models.json.bak", "chats.json.bak"} {
		if _, err := os.Stat(filepath.Join(storageDir, f)); err != nil {
			t.Fatalf("expected backup file %s to exist: %v", f, err)
		}
	}

	// Idempotency: running again against the already-migrated store must not
	// error and must not change the surviving counts.
	if err := MigrateDuplicateProviders(storageDir, logger); err != nil {
		t.Fatalf("second MigrateDuplicateProviders run failed: %v", err)
	}
	providersAgain := readArray(t, filepath.Join(storageDir, "providers.json"))
	if len(providersAgain) != 2 {
		t.Fatalf("expected provider count to stay at 2 after idempotent re-run, got %d", len(providersAgain))
	}
}

func TestMigrateDuplicateProviders_NoStorageDir(t *testing.T) {
	storageDir := t.TempDir() // empty, no providers.json at all
	if err := MigrateDuplicateProviders(storageDir, zap.NewNop()); err != nil {
		t.Fatalf("expected no error for a fresh install with no providers.json, got %v", err)
	}
}

func readArray(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("failed to unmarshal %s: %v", path, err)
	}
	return arr
}
