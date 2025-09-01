package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/drujensen/aiagent/internal/domain/services"

	"go.uber.org/zap"
)

type AgentCallTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger

	agentRepo    interfaces.AgentRepository
	chatService  services.ChatService
	sessionStore map[string]*AgentSession
	sessionMutex sync.RWMutex
}

type AgentSession struct {
	ID          string
	ParentAgent string
	Subagent    string
	TaskID      string
	Status      string // pending, active, completed, failed
	CreatedAt   time.Time
	CompletedAt *time.Time
	Result      any
	Error       error
	Context     map[string]any
}

type AgentCallRequest struct {
	AgentID string         `json:"agent_id"`
	Task    string         `json:"task"`
	Context map[string]any `json:"context,omitempty"`
	Tools   []string       `json:"tools,omitempty"`
	Timeout int            `json:"timeout,omitempty"` // minutes
}

type AgentCallResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Result    any    `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

type SubagentResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Result    any    `json:"result,omitempty"`
	Error     string `json:"error,omitempty"`
}

func NewAgentCallTool(name, description string, configuration map[string]string, logger *zap.Logger, agentRepo interfaces.AgentRepository, chatService services.ChatService) *AgentCallTool {
	return &AgentCallTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		agentRepo:     agentRepo,
		chatService:   chatService,
		sessionStore:  make(map[string]*AgentSession),
	}
}

// InjectDependencies allows injecting dependencies after tool creation
func (t *AgentCallTool) InjectDependencies(agentRepo interfaces.AgentRepository, chatService any) {
	t.agentRepo = agentRepo
	if cs, ok := chatService.(services.ChatService); ok {
		t.chatService = cs
	}
}

func (t *AgentCallTool) Name() string {
	return t.name
}

func (t *AgentCallTool) Description() string {
	return t.description
}

func (t *AgentCallTool) Configuration() map[string]string {
	return t.configuration
}

func (t *AgentCallTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *AgentCallTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.Description() + "\n\n")
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *AgentCallTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "agent_id",
			Type:        "string",
			Description: "ID of the agent to invoke",
			Required:    true,
		},
		{
			Name:        "task",
			Type:        "string",
			Description: "Detailed task description for the agent",
			Required:    true,
		},
		{
			Name:        "context",
			Type:        "object",
			Description: "Additional context data for the agent (optional)",
			Required:    false,
		},
		{
			Name:        "tools",
			Type:        "array",
			Description: "Specific tools to enable for this agent (optional)",
			Required:    false,
		},
		{
			Name:        "timeout",
			Type:        "integer",
			Description: "Maximum execution time in minutes (optional, default: 30)",
			Required:    false,
		},
	}
}

func (t *AgentCallTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing agent call", zap.String("arguments", arguments))

	// Check if dependencies are available
	if t.agentRepo == nil || t.chatService == nil {
		return "", fmt.Errorf("agent call tool dependencies not initialized - agent repository and chat service are required")
	}

	var req AgentCallRequest
	if err := json.Unmarshal([]byte(arguments), &req); err != nil {
		t.logger.Error("Failed to parse agent call request", zap.Error(err), zap.String("arguments", arguments))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if req.AgentID == "" {
		return "", fmt.Errorf("agent_id is required")
	}

	if req.Task == "" {
		return "", fmt.Errorf("task is required")
	}

	if req.Timeout <= 0 {
		req.Timeout = 30 // default 30 minutes
	}

	// Create new session
	sessionID := t.generateSessionID()
	session := &AgentSession{
		ID:          sessionID,
		ParentAgent: t.getCurrentAgentID(), // This would need to be passed in or retrieved from context
		Subagent:    req.AgentID,
		TaskID:      "", // Could be linked to task management system
		Status:      "pending",
		CreatedAt:   time.Now(),
		Context:     req.Context,
	}

	t.sessionMutex.Lock()
	t.sessionStore[sessionID] = session
	t.sessionMutex.Unlock()

	// Start agent execution asynchronously
	go t.executeAgent(session, req)

	response := AgentCallResponse{
		SessionID: sessionID,
		Status:    "pending",
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	return string(responseBytes), nil
}

func (t *AgentCallTool) executeAgent(session *AgentSession, req AgentCallRequest) {
	t.logger.Info("Starting subagent execution",
		zap.String("session_id", session.ID),
		zap.String("subagent", session.Subagent),
		zap.String("task", req.Task))

	// Update session status
	t.updateSessionStatus(session.ID, "active")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Minute)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("Subagent execution panicked", zap.Any("panic", r), zap.String("session_id", session.ID))
			t.completeSession(session.ID, nil, fmt.Errorf("subagent panicked: %v", r))
		}
	}()

	// Get subagent configuration
	_, err := t.agentRepo.GetAgent(ctx, req.AgentID)
	if err != nil {
		t.completeSession(session.ID, nil, fmt.Errorf("failed to get subagent: %v", err))
		return
	}

	// Create chat session for agent communication
	chat, err := t.chatService.CreateChat(ctx, req.AgentID, fmt.Sprintf("Subagent Session: %s", session.ID))
	if err != nil {
		t.completeSession(session.ID, nil, fmt.Errorf("failed to create chat session: %v", err))
		return
	}

	// Send initial task message to subagent
	taskMessage := fmt.Sprintf("Task: %s\n\nContext: %v", req.Task, req.Context)
	message := &entities.Message{
		Role:      "user",
		Content:   taskMessage,
		Timestamp: time.Now(),
	}

	_, err = t.chatService.SendMessage(ctx, chat.ID, message)
	if err != nil {
		t.completeSession(session.ID, nil, fmt.Errorf("failed to send task message: %v", err))
		return
	}

	// For now, we'll simulate subagent processing
	// In a real implementation, this would involve:
	// 1. Starting the subagent with the chat session
	// 2. Monitoring for completion
	// 3. Retrieving results

	time.Sleep(2 * time.Second) // Simulate processing time

	// Mock successful completion
	result := map[string]any{
		"status":          "completed",
		"output":          "Subagent task completed successfully",
		"chat_id":         chat.ID,
		"processing_time": "2s",
	}

	t.completeSession(session.ID, result, nil)
}

func (t *AgentCallTool) GetSessionResult(sessionID string) (*AgentCallResponse, error) {
	t.sessionMutex.RLock()
	session, exists := t.sessionStore[sessionID]
	t.sessionMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	response := &AgentCallResponse{
		SessionID: sessionID,
		Status:    session.Status,
		Result:    session.Result,
	}

	if session.Error != nil {
		response.Error = session.Error.Error()
	}

	return response, nil
}

func (t *AgentCallTool) ListAvailableAgents(ctx context.Context) ([]map[string]any, error) {
	agents, err := t.agentRepo.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %v", err)
	}

	var result []map[string]any
	for _, agent := range agents {
		result = append(result, map[string]any{
			"id":          agent.ID,
			"name":        agent.Name,
			"description": agent.Description,
			"tools":       agent.Tools,
			"model":       agent.Model,
		})
	}

	return result, nil
}

func (t *AgentCallTool) updateSessionStatus(sessionID, status string) {
	t.sessionMutex.Lock()
	defer t.sessionMutex.Unlock()

	if session, exists := t.sessionStore[sessionID]; exists {
		session.Status = status
		if status == "completed" || status == "failed" {
			now := time.Now()
			session.CompletedAt = &now
		}
	}
}

func (t *AgentCallTool) completeSession(sessionID string, result any, err error) {
	t.sessionMutex.Lock()
	defer t.sessionMutex.Unlock()

	if session, exists := t.sessionStore[sessionID]; exists {
		session.Result = result
		session.Error = err
		if err != nil {
			session.Status = "failed"
		} else {
			session.Status = "completed"
		}
		now := time.Now()
		session.CompletedAt = &now
	}
}

func (t *AgentCallTool) generateSessionID() string {
	return fmt.Sprintf("subagent_%d", time.Now().UnixNano())
}

func (t *AgentCallTool) getCurrentAgentID() string {
	// This would need to be implemented to get the current agent ID from context
	// For now, return a placeholder
	return "current-agent"
}

func (t *AgentCallTool) CleanupCompletedSessions() {
	t.sessionMutex.Lock()
	defer t.sessionMutex.Unlock()

	// Remove sessions older than 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)
	for id, session := range t.sessionStore {
		if session.CompletedAt != nil && session.CompletedAt.Before(cutoff) {
			delete(t.sessionStore, id)
		}
	}
}

var _ entities.Tool = (*AgentCallTool)(nil)
