package repositories_json

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"go.uber.org/zap"
)

// MigrateDuplicateProviders collapses provider entries that share the same
// Name (a duplicate-seeding bug fixed in defaults.go), re-points any models
// that referenced a removed provider ID, collapses any resulting duplicate
// (provider_id, model_name) model pairs, and re-points any chats that
// referenced a removed model ID. It is idempotent: if no duplicates are
// found, it makes no changes and writes no backup files.
func MigrateDuplicateProviders(storageDir string, logger *zap.Logger) error {
	providersPath := filepath.Join(storageDir, "providers.json")
	modelsPath := filepath.Join(storageDir, "models.json")
	chatsPath := filepath.Join(storageDir, "chats.json")

	providers, err := readJSONArray(providersPath)
	if os.IsNotExist(err) || providers == nil {
		return nil // fresh install, nothing to migrate
	}
	if err != nil {
		return err
	}

	removedProviderIDs := map[string]string{} // removedID -> survivingID
	byName := map[string][]map[string]any{}
	for _, p := range providers {
		name, _ := p["name"].(string)
		byName[name] = append(byName[name], p)
	}

	var dedupedProviders []map[string]any
	providersChanged := false
	for name, group := range byName {
		if len(group) < 2 {
			dedupedProviders = append(dedupedProviders, group...)
			continue
		}
		ids := make([]string, 0, len(group))
		byID := map[string]map[string]any{}
		for _, p := range group {
			id, _ := p["id"].(string)
			ids = append(ids, id)
			byID[id] = p
		}
		sort.Strings(ids) // lowest UUID lexicographically survives, deterministic tie-break
		survivor := ids[0]
		dedupedProviders = append(dedupedProviders, byID[survivor])
		for _, id := range ids[1:] {
			removedProviderIDs[id] = survivor
		}
		providersChanged = true
		logger.Info("migrated duplicate provider entries",
			zap.String("name", name),
			zap.String("survivingID", survivor),
			zap.Strings("removedIDs", ids[1:]))
	}

	if !providersChanged {
		return nil
	}

	if err := backupFile(providersPath); err != nil {
		return err
	}
	if err := writeJSONArray(providersPath, dedupedProviders); err != nil {
		return err
	}

	// Re-point models that referenced a removed provider, then collapse any
	// resulting duplicate (provider_id, model_name) pairs.
	models, err := readJSONArray(modelsPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if models != nil {
		modelsChanged := false
		for _, m := range models {
			pid, _ := m["provider_id"].(string)
			if survivor, ok := removedProviderIDs[pid]; ok {
				m["provider_id"] = survivor
				modelsChanged = true
			}
		}

		removedModelIDs := map[string]string{} // removedID -> survivingID
		type key struct{ providerID, modelName string }
		byKey := map[key][]map[string]any{}
		for _, m := range models {
			pid, _ := m["provider_id"].(string)
			mn, _ := m["model_name"].(string)
			k := key{pid, mn}
			byKey[k] = append(byKey[k], m)
		}

		var dedupedModels []map[string]any
		for _, group := range byKey {
			if len(group) < 2 {
				dedupedModels = append(dedupedModels, group...)
				continue
			}
			ids := make([]string, 0, len(group))
			byID := map[string]map[string]any{}
			for _, m := range group {
				id, _ := m["id"].(string)
				ids = append(ids, id)
				byID[id] = m
			}
			sort.Strings(ids)
			survivor := ids[0]
			dedupedModels = append(dedupedModels, byID[survivor])
			for _, id := range ids[1:] {
				removedModelIDs[id] = survivor
			}
			modelsChanged = true
		}

		if modelsChanged {
			if err := backupFile(modelsPath); err != nil {
				return err
			}
			if err := writeJSONArray(modelsPath, dedupedModels); err != nil {
				return err
			}
			logger.Info("collapsed duplicate models after provider migration", zap.Int("removedModelCount", len(removedModelIDs)))

			if len(removedModelIDs) > 0 {
				chats, err := readJSONArray(chatsPath)
				if err != nil && !os.IsNotExist(err) {
					return err
				}
				if chats != nil {
					chatsChanged := false
					for _, c := range chats {
						mid, _ := c["model_id"].(string)
						if survivor, ok := removedModelIDs[mid]; ok {
							c["model_id"] = survivor
							chatsChanged = true
						}
					}
					if chatsChanged {
						if err := backupFile(chatsPath); err != nil {
							return err
						}
						if err := writeJSONArray(chatsPath, chats); err != nil {
							return err
						}
						logger.Info("re-pointed chats referencing removed models")
					}
				}
			}
		}
	}

	return nil
}

func readJSONArray(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func writeJSONArray(path string, arr []map[string]any) error {
	data, err := json.MarshalIndent(arr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path+".bak", data, 0644)
}
