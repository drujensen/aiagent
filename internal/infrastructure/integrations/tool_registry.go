package integrations

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/infrastructure/integrations/tools"
)

var predefinedTools = []struct {
	name     string
	category string
	factory  func(workspace string) interfaces.ToolIntegration
}{
	{
		name:     "Search",
		category: "search",
		factory:  func(_ string) interfaces.ToolIntegration { return tools.NewSearchTool() },
	},
	{
		name:     "Bash",
		category: "bash",
		factory:  func(workspace string) interfaces.ToolIntegration { return tools.NewBashTool(workspace) },
	},
	{
		name:     "File",
		category: "file",
		factory:  func(workspace string) interfaces.ToolIntegration { return tools.NewFileTool(workspace) },
	},
}

var toolInstancesByID map[string]interfaces.ToolIntegration

func InitializeTools(ctx context.Context, repo interfaces.ToolRepository, workspace string) error {
	toolInstancesByID = make(map[string]interfaces.ToolIntegration)

	for _, pt := range predefinedTools {
		existingTools, err := repo.ListTools(ctx)
		if err != nil {
			return err
		}

		var toolEntity *entities.Tool
		for _, t := range existingTools {
			if t.Name == pt.name {
				toolEntity = t
				break
			}
		}

		if toolEntity == nil {
			toolEntity = &entities.Tool{
				Name:        pt.name,
				Description: pt.category,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			if err := repo.CreateTool(ctx, toolEntity); err != nil {
				return err
			}
		}

		toolInstance := pt.factory(workspace)
		toolInstancesByID[toolEntity.ID.Hex()] = toolInstance
	}

	return nil
}

func GetToolByID(id string) interfaces.ToolIntegration {
	return toolInstancesByID[id]
}
