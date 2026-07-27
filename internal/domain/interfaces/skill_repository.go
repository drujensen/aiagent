package interfaces

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

type SkillRepository interface {
	DiscoverSkills(ctx context.Context) ([]*entities.Skill, error)
	Save(skill *entities.Skill) error

	// MaterializeBuiltinSkills copies the repository's embedded builtin
	// skills into the managed ~/.aiagent/skills/aiagent-builtin/ subdirectory,
	// always overwriting so it exactly matches the embedded version. Call
	// this once at startup before DiscoverSkills is relied upon.
	MaterializeBuiltinSkills() error
}
