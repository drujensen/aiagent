package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/drujensen/aiagent/internal/impl/repositories"

	"github.com/stretchr/testify/assert"
)

// TestMaterializeBuiltinSkills_FreshHomeMatchesEmbedded is AC6c's
// deliverable test: it overrides HOME to a fresh temp directory (never
// touching the developer's real ~/.aiagent/skills/ or any directory that
// might already contain materialized skills from a prior run), runs
// MaterializeBuiltinSkills against the real embedded skills/ tree, and
// asserts the materialized file bytes match the embedded source exactly -
// not merely that materialization "succeeded".
func TestMaterializeBuiltinSkills_FreshHomeMatchesEmbedded(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	skillRepo := repositories.NewSkillRepository(embeddedSkillsFS)
	if err := skillRepo.MaterializeBuiltinSkills(); err != nil {
		t.Fatalf("MaterializeBuiltinSkills failed: %v", err)
	}

	fileCount := 0
	err := fs.WalkDir(embeddedSkillsFS, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fileCount++

		rel, err := filepath.Rel("skills", path)
		if err != nil {
			return err
		}
		materializedPath := filepath.Join(tmpHome, ".aiagent", "skills", "aiagent-builtin", rel)

		want, err := embeddedSkillsFS.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read embedded file %s: %v", path, err)
		}
		got, err := os.ReadFile(materializedPath)
		if err != nil {
			t.Fatalf("materialized file missing at %s: %v", materializedPath, err)
		}
		assert.Equal(t, want, got, "materialized bytes for %s must match the embedded source", rel)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk embedded skills/: %v", err)
	}
	if fileCount == 0 {
		t.Fatal("precondition failed: embedded skills/ tree has zero files - the test cannot be meaningful")
	}

	// Also confirm all 3 pipeline skills are discoverable through the real
	// DiscoverSkills path (not just that files were written), against this
	// same fresh HOME.
	skills, err := skillRepo.DiscoverSkills(context.Background())
	if err != nil {
		t.Fatalf("DiscoverSkills failed: %v", err)
	}
	found := make(map[string]bool, len(skills))
	for _, s := range skills {
		found[s.Name] = true
	}
	for _, name := range []string{"research", "design", "plan"} {
		if !found[name] {
			t.Errorf("expected skill %q to be discoverable after materialization, got skills: %v", name, found)
		}
	}
}
