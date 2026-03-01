package repositories

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"gopkg.in/yaml.v3"
)

// SkillFrontmatter represents the YAML frontmatter in SKILL.md
type SkillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
	AllowedTools  []string          `yaml:"allowed-tools,omitempty"`
}

type SkillRepository struct{}

func NewSkillRepository() interfaces.SkillRepository {
	return &SkillRepository{}
}

func (r *SkillRepository) DiscoverSkills(ctx context.Context) ([]*entities.Skill, error) {
	skillMap := make(map[string]*entities.Skill)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.InternalErrorf("failed to get home directory: %v", err)
	}

	// Get current working directory for project skills
	cwd, cwdErr := os.Getwd()

	// Scan skill directories in priority order
	skillDirs := []struct {
		path      string
		isProject bool
	}{}
	if cwdErr == nil {
		skillDirs = append(skillDirs, struct {
			path      string
			isProject bool
		}{filepath.Join(cwd, ".aiagent", "skills"), true})
	} else {
		fmt.Printf("Warning: failed to get current directory for project skills: %v\n", cwdErr)
	}
	skillDirs = append(skillDirs,
		struct {
			path      string
			isProject bool
		}{filepath.Join(homeDir, ".aiagent", "skills"), false},
		struct {
			path      string
			isProject bool
		}{filepath.Join(homeDir, ".agents", "skills"), false},
	)

	for _, dir := range skillDirs {
		if err := r.scanSkillsDirectory(dir.path, skillMap, dir.isProject); err != nil {
			// Log warning but continue - don't fail on inaccessible directories
			fmt.Printf("Warning: failed to scan %s: %v\n", dir.path, err)
		}
	}

	// Convert map to slice and sort by name
	var skills []*entities.Skill
	for _, skill := range skillMap {
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Name < skills[j].Name
	})

	return skills, nil
}

func (r *SkillRepository) scanSkillsDirectory(dir string, skillMap map[string]*entities.Skill, isProject bool) error {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, skip
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name())
		skillMdPath := filepath.Join(skillPath, "SKILL.md")

		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			continue // No SKILL.md, skip
		}

		skill, err := r.parseSkillFile(skillMdPath, skillPath)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", skillMdPath, err)
			continue
		}

		// Check if skill already exists
		existing, exists := skillMap[skill.Name]
		if !exists {
			skillMap[skill.Name] = skill
		} else {
			// Prefer project over home, but don't overwrite if both are project or both home
			existingIsProject := strings.HasPrefix(existing.Path, filepath.Dir(skillPath))
			if isProject && !existingIsProject {
				skillMap[skill.Name] = skill
			}
		}
	}

	return nil
}

func (r *SkillRepository) parseSkillFile(filePath, skillPath string) (*entities.Skill, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.InternalErrorf("failed to read file: %v", err)
	}

	content := string(data)

	// Find YAML frontmatter (between ---)
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid SKILL.md format: missing YAML frontmatter")
	}

	var frontmatter SkillFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %v", err)
	}

	skill := &entities.Skill{
		Name:          frontmatter.Name,
		Summary:       frontmatter.Description,
		Path:          skillPath,
		License:       frontmatter.License,
		Compatibility: frontmatter.Compatibility,
		Metadata:      frontmatter.Metadata,
		AllowedTools:  frontmatter.AllowedTools,
		// Content will be loaded later
	}

	if err := skill.Validate(); err != nil {
		return nil, fmt.Errorf("skill validation failed: %v", err)
	}

	return skill, nil
}

// Save saves a skill to the repository by creating the directory and writing SKILL.md
func (r *SkillRepository) Save(skill *entities.Skill) error {
	if err := skill.Validate(); err != nil {
		return fmt.Errorf("skill validation failed: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	skillDir := filepath.Join(homeDir, ".aiagent", "skills", skill.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	// Build frontmatter
	frontmatter := fmt.Sprintf("---\nname: %s\ndescription: %s\n", skill.Name, skill.Summary)
	if skill.License != "" {
		frontmatter += fmt.Sprintf("license: %s\n", skill.License)
	}
	if skill.Compatibility != "" {
		frontmatter += fmt.Sprintf("compatibility: %s\n", skill.Compatibility)
	}
	if len(skill.AllowedTools) > 0 {
		frontmatter += "allowed-tools:\n"
		for _, tool := range skill.AllowedTools {
			frontmatter += fmt.Sprintf("  - %s\n", tool)
		}
	}
	if len(skill.Metadata) > 0 {
		frontmatter += "metadata:\n"
		for k, v := range skill.Metadata {
			frontmatter += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}
	frontmatter += "---\n"

	content := frontmatter + skill.Content

	if err := os.WriteFile(skillMdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write SKILL.md: %w", err)
	}

	return nil
}
