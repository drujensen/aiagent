package services

import (
	"context"
	"fmt"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"
)

/**
 * @description
 * This file implements the AgentService for the AI Workflow Automation Platform.
 * It manages the lifecycle of AIAgent entities, including creation, updates, deletion, and retrieval,
 * as well as hierarchy validation to maintain a tree-like structure. The service acts as the business
 * logic layer, enforcing domain rules and interacting with the AgentRepository for persistence.
 *
 * Key features:
 * - CRUD Operations: Handles creation, updating, deletion, and retrieval of agents.
 * - Hierarchy Management: Validates parent-child relationships to prevent cycles and ensure consistency.
 * - Configuration Validation: Ensures required fields and settings are present before persistence.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: Provides the AIAgent entity definition.
 * - aiagent/internal/domain/repositories: Provides the AgentRepository interface and ErrNotFound.
 *
 * @notes
 * - Validation occurs before repository operations to prevent invalid data persistence.
 * - Hierarchy cycles are checked using a recursive traversal method.
 * - Edge cases include missing required fields, invalid parent IDs, and duplicate agent names.
 * - Assumption: Agent IDs are strings matching MongoDB ObjectID hex format.
 * - Limitation: No explicit locking for concurrent hierarchy updates; relies on MongoDB's atomicity.
 */

// AgentService defines the interface for managing AIAgent entities in the domain layer.
// It provides methods for CRUD operations and hierarchy validation, abstracting the implementation.
type AgentService interface {
	// CreateAgent creates a new agent with the provided configuration.
	// Validates required fields and hierarchy before persisting via the repository.
	// Returns an error if validation fails or persistence encounters an issue.
	CreateAgent(ctx context.Context, agent *entities.AIAgent) error

	// UpdateAgent updates an existing agent's configuration and hierarchy.
	// Validates the updated agent and hierarchy before persisting changes.
	// Returns an error if the agent doesn't exist, validation fails, or update fails.
	UpdateAgent(ctx context.Context, agent *entities.AIAgent) error

	// DeleteAgent removes an agent by its ID, ensuring no child agents remain.
	// Returns an error if the agent doesn't exist, has children, or deletion fails.
	DeleteAgent(ctx context.Context, id string) error

	// GetAgent retrieves an agent by its ID.
	// Returns the agent and nil error on success, or nil and an error if not found or retrieval fails.
	GetAgent(ctx context.Context, id string) (*entities.AIAgent, error)

	// ListAgents retrieves all agents in the system.
	// Returns a slice of agents, empty if none exist, or an error if retrieval fails.
	ListAgents(ctx context.Context) ([]*entities.AIAgent, error)

	// ValidateHierarchy ensures the agent's hierarchy is valid (e.g., no cycles, valid parent).
	// Returns an error if the hierarchy is invalid (e.g., cycle detected, parent not found).
	ValidateHierarchy(ctx context.Context, agent *entities.AIAgent) error
}

// agentService implements the AgentService interface.
// It uses an AgentRepository for persistence and enforces business rules.
type agentService struct {
	agentRepo repositories.AgentRepository
}

// NewAgentService creates a new instance of agentService with the given repository.
//
// Parameters:
// - agentRepo: The repository for managing AIAgent entities.
//
// Returns:
// - *agentService: A new instance implementing AgentService.
func NewAgentService(agentRepo repositories.AgentRepository) *agentService {
	return &agentService{
		agentRepo: agentRepo,
	}
}

// CreateAgent creates a new agent after validating its configuration and hierarchy.
// Sets initial timestamps and persists the agent via the repository.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - agent: Pointer to the AIAgent entity to create; its ID is set upon success.
//
// Returns:
// - error: Nil on success, or an error if validation or persistence fails.
func (s *agentService) CreateAgent(ctx context.Context, agent *entities.AIAgent) error {
	// Validate required fields
	if agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if agent.Prompt == "" {
		return fmt.Errorf("agent prompt is required")
	}
	if agent.Configuration == nil {
		return fmt.Errorf("agent configuration is required")
	}
	if _, ok := agent.Configuration["provider"]; !ok {
		return fmt.Errorf("agent configuration must specify a provider")
	}

	// Validate optional fields with defaults
	if agent.Tools == nil {
		agent.Tools = []string{}
	}
	if agent.ChildrenIDs == nil {
		agent.ChildrenIDs = []string{}
	}
	if agent.HumanInteractionEnabled == false {
		agent.HumanInteractionEnabled = true // Default per spec
	}

	// Validate hierarchy
	if err := s.ValidateHierarchy(ctx, agent); err != nil {
		return fmt.Errorf("hierarchy validation failed: %w", err)
	}

	// Set timestamps
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	// Persist the agent
	if err := s.agentRepo.CreateAgent(ctx, agent); err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	return nil
}

// UpdateAgent updates an existing agent's details after validation.
// Ensures the agent exists and hierarchy remains valid post-update.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - agent: Pointer to the AIAgent entity with updated fields; must have a valid ID.
//
// Returns:
// - error: Nil on success, or an error if validation, existence check, or update fails.
func (s *agentService) UpdateAgent(ctx context.Context, agent *entities.AIAgent) error {
	if agent.ID == "" {
		return fmt.Errorf("agent ID is required for update")
	}

	// Check if agent exists
	existing, err := s.agentRepo.GetAgent(ctx, agent.ID)
	if err != nil {
		if err == repositories.ErrNotFound {
			return fmt.Errorf("agent not found: %s", agent.ID)
		}
		return fmt.Errorf("failed to retrieve agent: %w", err)
	}

	// Validate required fields
	if agent.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	if agent.Prompt == "" {
		return fmt.Errorf("agent prompt is required")
	}
	if agent.Configuration == nil {
		return fmt.Errorf("agent configuration is required")
	}
	if _, ok := agent.Configuration["provider"]; !ok {
		return fmt.Errorf("agent configuration must specify a provider")
	}

	// Preserve timestamps unless explicitly overridden; update UpdatedAt
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = existing.CreatedAt
	}
	agent.UpdatedAt = time.Now()

	// Validate optional fields with defaults
	if agent.Tools == nil {
		agent.Tools = []string{}
	}
	if agent.ChildrenIDs == nil {
		agent.ChildrenIDs = []string{}
	}

	// Validate hierarchy
	if err := s.ValidateHierarchy(ctx, agent); err != nil {
		return fmt.Errorf("hierarchy validation failed: %w", err)
	}

	// Persist the update
	if err := s.agentRepo.UpdateAgent(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

// DeleteAgent removes an agent by ID, ensuring it has no child agents.
// Updates parent agent's ChildrenIDs if applicable.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the agent to delete.
//
// Returns:
// - error: Nil on success, or an error if agent doesn’t exist, has children, or deletion fails.
func (s *agentService) DeleteAgent(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("agent ID is required for deletion")
	}

	agent, err := s.agentRepo.GetAgent(ctx, id)
	if err != nil {
		if err == repositories.ErrNotFound {
			return fmt.Errorf("agent not found: %s", id)
		}
		return fmt.Errorf("failed to retrieve agent: %w", err)
	}

	// Prevent deletion if agent has children
	if len(agent.ChildrenIDs) > 0 {
		return fmt.Errorf("cannot delete agent with %d child agents", len(agent.ChildrenIDs))
	}

	// Remove agent from parent's ChildrenIDs if applicable
	if agent.ParentID != "" {
		parent, err := s.agentRepo.GetAgent(ctx, agent.ParentID)
		if err != nil && err != repositories.ErrNotFound {
			return fmt.Errorf("failed to retrieve parent agent: %w", err)
		}
		if parent != nil {
			newChildren := make([]string, 0, len(parent.ChildrenIDs)-1)
			for _, childID := range parent.ChildrenIDs {
				if childID != id {
					newChildren = append(newChildren, childID)
				}
			}
			parent.ChildrenIDs = newChildren
			if err := s.agentRepo.UpdateAgent(ctx, parent); err != nil {
				return fmt.Errorf("failed to update parent agent: %w", err)
			}
		}
	}

	// Delete the agent
	if err := s.agentRepo.DeleteAgent(ctx, id); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}

// GetAgent retrieves an agent by its ID from the repository.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - id: The string ID of the agent to retrieve.
//
// Returns:
// - *entities.AIAgent: The retrieved agent, or nil if not found.
// - error: Nil on success, ErrNotFound if agent doesn’t exist, or another error otherwise.
func (s *agentService) GetAgent(ctx context.Context, id string) (*entities.AIAgent, error) {
	if id == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	agent, err := s.agentRepo.GetAgent(ctx, id)
	if err != nil {
		return nil, err // ErrNotFound or other errors propagated from repository
	}

	return agent, nil
}

// ListAgents retrieves all agents from the repository.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
//
// Returns:
// - []*entities.AIAgent: Slice of all agents, empty if none exist.
// - error: Nil on success, or an error if retrieval fails.
func (s *agentService) ListAgents(ctx context.Context) ([]*entities.AIAgent, error) {
	agents, err := s.agentRepo.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return agents, nil
}

// ValidateHierarchy checks the agent's hierarchy for validity.
// Ensures no cycles exist and the parent ID, if set, references an existing agent.
//
// Parameters:
// - ctx: Context for timeout and cancellation.
// - agent: Pointer to the AIAgent entity to validate.
//
// Returns:
// - error: Nil if hierarchy is valid, or an error if invalid (e.g., cycle, missing parent).
func (s *agentService) ValidateHierarchy(ctx context.Context, agent *entities.AIAgent) error {
	if agent.ParentID == "" && len(agent.ChildrenIDs) == 0 {
		return nil // No hierarchy to validate
	}

	// Check if ParentID exists
	if agent.ParentID != "" {
		parent, err := s.agentRepo.GetAgent(ctx, agent.ParentID)
		if err != nil {
			if err == repositories.ErrNotFound {
				return fmt.Errorf("parent agent not found: %s", agent.ParentID)
			}
			return fmt.Errorf("failed to retrieve parent agent: %w", err)
		}
		// Prevent self-reference
		if parent.ID == agent.ID {
			return fmt.Errorf("agent cannot be its own parent")
		}
	}

	// Check for cycles by traversing up the hierarchy
	visited := make(map[string]bool)
	currentID := agent.ID
	for currentID != "" {
		if visited[currentID] {
			return fmt.Errorf("hierarchy cycle detected at agent: %s", currentID)
		}
		visited[currentID] = true

		currentAgent, err := s.agentRepo.GetAgent(ctx, currentID)
		if err != nil {
			if err == repositories.ErrNotFound {
				break // Shouldn’t happen due to prior validation, but safe exit
			}
			return fmt.Errorf("failed to traverse hierarchy: %w", err)
		}
		currentID = currentAgent.ParentID
	}

	// Validate children exist (optional check, as children may be added later)
	for _, childID := range agent.ChildrenIDs {
		_, err := s.agentRepo.GetAgent(ctx, childID)
		if err != nil {
			if err == repositories.ErrNotFound {
				return fmt.Errorf("child agent not found: %s", childID)
			}
			return fmt.Errorf("failed to validate child agent: %w", err)
		}
	}

	return nil
}
