package interfaces

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type SkillRepository interface {
	DiscoverSkills(ctx context.Context) ([]*entities.Skill, error)
	Save(skill *entities.Skill) error
}
