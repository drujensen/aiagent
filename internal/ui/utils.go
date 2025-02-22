package ui

import (
	"aiagent/internal/domain/entities"
)

// AgentNode represents a node in the agent hierarchy tree for UI rendering.
type AgentNode struct {
	ID       string       // Unique identifier of the agent
	Name     string       // Display name of the agent
	Children []*AgentNode // Recursive list of child nodes
}

// buildHierarchy constructs a tree of AgentNodes from a flat list of AIAgents.
// It maps agents by ID and recursively builds parent-child relationships.
//
// Parameters:
// - agents: List of AIAgent entities from the service.
//
// Returns:
// - []*AgentNode: List of root nodes (agents with no parent).
func buildHierarchy(agents []*entities.AIAgent) []*AgentNode {
	agentMap := make(map[string]*entities.AIAgent)
	for _, agent := range agents {
		agentMap[agent.ID] = agent
	}
	rootAgents := []*AgentNode{}
	for _, agent := range agents {
		if agent.ParentID == "" {
			node := &AgentNode{ID: agent.ID, Name: agent.Name}
			rootAgents = append(rootAgents, node)
			buildChildren(node, agentMap)
		}
	}
	return rootAgents
}

// buildChildren recursively populates the Children field of an AgentNode.
// It looks up child agents in the map and builds their subtrees.
//
// Parameters:
// - node: Current node being processed.
// - agentMap: Map of agent IDs to AIAgent entities for quick lookup.
func buildChildren(node *AgentNode, agentMap map[string]*entities.AIAgent) {
	agent := agentMap[node.ID]
	for _, childID := range agent.ChildrenIDs {
		if childAgent, exists := agentMap[childID]; exists {
			childNode := &AgentNode{ID: childAgent.ID, Name: childAgent.Name}
			node.Children = append(node.Children, childNode)
			buildChildren(childNode, agentMap)
		}
	}
}
