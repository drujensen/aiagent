package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"go.uber.org/zap"
)

type SkillService interface {
	ListSkills(ctx context.Context) ([]*entities.Skill, error)
	GetSkillContent(ctx context.Context, skillName string) (string, error)
}

type skillService struct {
	repo   interfaces.SkillRepository
	logger *zap.Logger
}

func NewSkillService(repo interfaces.SkillRepository, logger *zap.Logger) SkillService {
	return &skillService{
		repo:   repo,
		logger: logger,
	}
}

func (s *skillService) ListSkills(ctx context.Context) ([]*entities.Skill, error) {
	return s.repo.DiscoverSkills(ctx)
}

func (s *skillService) GetSkillContent(ctx context.Context, skillName string) (string, error) {
	skills, err := s.ListSkills(ctx)
	if err != nil {
		return "", err
	}

	var skill *entities.Skill
	for _, sk := range skills {
		if sk.Name == skillName {
			skill = sk
			break
		}
	}
	if skill == nil {
		return "", errors.NotFoundErrorf("skill %s not found", skillName)
	}

	skillMdPath := filepath.Join(skill.Path, "SKILL.md")
	data, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", errors.InternalErrorf("failed to read skill file: %v", err)
	}

	content := string(data)

	// Load referenced files on demand
	content, err = s.loadReferencedFiles(content, skill.Path)
	if err != nil {
		s.logger.Warn("Failed to load some referenced files", zap.Error(err))
		// Continue with original content
	}

	return content, nil
}

func (s *skillService) loadReferencedFiles(content, skillPath string) (string, error) {
	// Simple implementation: look for relative file references and inline them
	// This is a basic version; could be enhanced to handle different file types

	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Check for markdown links like [text](references/file.md)
		if strings.Contains(line, "](") && strings.Contains(line, ")") {
			// Basic regex-free parsing for relative paths
			start := strings.Index(line, "](")
			if start != -1 {
				end := strings.Index(line[start+2:], ")")
				if end != -1 {
					refPath := line[start+2 : start+2+end]
					if !strings.HasPrefix(refPath, "http") && !filepath.IsAbs(refPath) {
						fullPath := filepath.Join(skillPath, refPath)
						if data, err := os.ReadFile(fullPath); err == nil {
							// Inline the file content
							result = append(result, line)
							result = append(result, "```")
							result = append(result, string(data))
							result = append(result, "```")
							continue
						}
					}
				}
			}
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n"), nil
}
