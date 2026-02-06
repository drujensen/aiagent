package entities

import (
	"testing"
	"time"
)

func TestNewAgent(t *testing.T) {
	name := "TestAgent"
	systemPrompt := "You are a test agent."
	tools := []string{"FileRead", "FileWrite"}

	agent := NewAgent(name, systemPrompt, tools)

	if agent.Name != name {
		t.Errorf("Expected name %s, got %s", name, agent.Name)
	}
	if agent.SystemPrompt != systemPrompt {
		t.Errorf("Expected system prompt %s, got %s", systemPrompt, agent.SystemPrompt)
	}
	if len(agent.Tools) != len(tools) {
		t.Errorf("Expected %d tools, got %d", len(tools), len(agent.Tools))
	}
	for i, tool := range tools {
		if agent.Tools[i] != tool {
			t.Errorf("Expected tool %s at index %d, got %s", tool, i, agent.Tools[i])
		}
	}
}

func TestAgent_FilterValue(t *testing.T) {
	agent := &Agent{
		Name: "TestAgent",
	}

	expected := "TestAgent"
	result := agent.FilterValue()

	if result != expected {
		t.Errorf("Expected filter value %s, got %s", expected, result)
	}
}

func TestAgent_Title(t *testing.T) {
	agent := &Agent{Name: "TestAgent"}

	if agent.Title() != "TestAgent" {
		t.Errorf("Expected title 'TestAgent', got %s", agent.Title())
	}
}

func TestAgent_Description(t *testing.T) {
	agent := &Agent{
		Name:  "TestAgent",
		Tools: []string{"FileRead", "FileWrite", "WebSearch"},
	}

	expected := "Tools: 3"
	result := agent.Description()

	if result != expected {
		t.Errorf("Expected description %s, got %s", expected, result)
	}
}

func TestAgent_FullSystemPrompt(t *testing.T) {
	systemPrompt := "You are a helpful assistant."
	agent := &Agent{
		Name:         "TestAgent",
		SystemPrompt: systemPrompt,
	}

	result := agent.FullSystemPrompt()

	if !contains(result, "Your name is TestAgent") {
		t.Errorf("Expected full system prompt to contain agent name, got %s", result)
	}
	if !contains(result, systemPrompt) {
		t.Errorf("Expected full system prompt to contain original prompt, got %s", result)
	}
	if !contains(result, "Current date and time is") {
		t.Errorf("Expected full system prompt to contain timestamp, got %s", result)
	}
}

func TestNewProvider(t *testing.T) {
	id := "test-id"
	name := "Test Provider"
	providerType := ProviderOpenAI
	baseURL := "https://api.openai.com"
	apiKeyName := "OPENAI_API_KEY"
	models := []ModelPricing{
		{Name: "gpt-4", InputPricePerMille: 30.00, OutputPricePerMille: 60.00, ContextWindow: 128000},
	}

	provider := NewProvider(id, name, providerType, baseURL, apiKeyName, models)

	if provider.ID != id {
		t.Errorf("Expected ID %s, got %s", id, provider.ID)
	}
	if provider.Name != name {
		t.Errorf("Expected name %s, got %s", name, provider.Name)
	}
	if provider.Type != providerType {
		t.Errorf("Expected type %v, got %v", providerType, provider.Type)
	}
	if provider.BaseURL != baseURL {
		t.Errorf("Expected base URL %s, got %s", baseURL, provider.BaseURL)
	}
	if provider.APIKeyName != apiKeyName {
		t.Errorf("Expected API key name %s, got %s", apiKeyName, provider.APIKeyName)
	}
	if len(provider.Models) != len(models) {
		t.Errorf("Expected %d models, got %d", len(models), len(provider.Models))
	}
}

func TestProvider_GetModelPricing(t *testing.T) {
	models := []ModelPricing{
		{Name: "gpt-4", InputPricePerMille: 30.00, OutputPricePerMille: 60.00, ContextWindow: 128000},
		{Name: "gpt-3.5", InputPricePerMille: 5.00, OutputPricePerMille: 15.00, ContextWindow: 64000},
	}

	provider := &Provider{Models: models}

	// Test existing model
	pricing := provider.GetModelPricing("gpt-4")
	if pricing == nil {
		t.Fatal("Expected pricing for gpt-4, got nil")
	}
	if pricing.InputPricePerMille != 30.00 {
		t.Errorf("Expected input price 30.00, got %f", pricing.InputPricePerMille)
	}

	// Test non-existing model
	pricing = provider.GetModelPricing("non-existent")
	if pricing != nil {
		t.Errorf("Expected nil for non-existent model, got %v", pricing)
	}
}

func TestNewChat(t *testing.T) {
	agentID := "test-agent-id"
	modelID := "test-model-id"
	name := "Test Chat"

	chat := NewChat(agentID, modelID, name)

	if chat.AgentID != agentID {
		t.Errorf("Expected agent ID %s, got %s", agentID, chat.AgentID)
	}
	if chat.ModelID != modelID {
		t.Errorf("Expected model ID %s, got %s", modelID, chat.ModelID)
	}
	if chat.Name != name {
		t.Errorf("Expected name %s, got %s", name, chat.Name)
	}
	if chat.Active != true {
		t.Errorf("Expected chat to be active by default")
	}
	if chat.Usage == nil {
		t.Errorf("Expected usage to be initialized")
	}
	if len(chat.Messages) != 0 {
		t.Errorf("Expected messages to be empty slice")
	}
}

func TestChat_UpdateUsage(t *testing.T) {
	chat := NewChat("test-agent", "test-model", "Test Chat")

	// Add messages with usage
	msg1 := &Message{
		Usage: &Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			Cost:             0.5,
		},
	}
	msg2 := &Message{
		Usage: &Usage{
			PromptTokens:     15,
			CompletionTokens: 25,
			Cost:             0.75,
		},
	}

	chat.Messages = []Message{*msg1, *msg2}
	chat.UpdateUsage()

	expectedPromptTokens := 25
	expectedCompletionTokens := 45
	expectedTotalTokens := 70
	expectedCost := 1.25

	if chat.Usage.TotalPromptTokens != expectedPromptTokens {
		t.Errorf("Expected total prompt tokens %d, got %d", expectedPromptTokens, chat.Usage.TotalPromptTokens)
	}
	if chat.Usage.TotalCompletionTokens != expectedCompletionTokens {
		t.Errorf("Expected total completion tokens %d, got %d", expectedCompletionTokens, chat.Usage.TotalCompletionTokens)
	}
	if chat.Usage.TotalTokens != expectedTotalTokens {
		t.Errorf("Expected total tokens %d, got %d", expectedTotalTokens, chat.Usage.TotalTokens)
	}
	if chat.Usage.TotalCost != expectedCost {
		t.Errorf("Expected total cost %f, got %f", expectedCost, chat.Usage.TotalCost)
	}
}

func TestChat_FilterValue(t *testing.T) {
	chat := &Chat{Name: "Test Chat"}

	if chat.FilterValue() != "Test Chat" {
		t.Errorf("Expected filter value 'Test Chat', got %s", chat.FilterValue())
	}
}

func TestChat_Title(t *testing.T) {
	chat := &Chat{Name: "Test Chat"}

	if chat.Title() != "Test Chat" {
		t.Errorf("Expected title 'Test Chat', got %s", chat.Title())
	}
}

func TestChat_Description(t *testing.T) {
	createdAt := time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC)
	chat := &Chat{
		CreatedAt: createdAt,
	}

	expected := "2023-01-15 10:30"
	result := chat.Description()

	if result != expected {
		t.Errorf("Expected description %s, got %s", expected, result)
	}
}

func TestNewTask(t *testing.T) {
	name := "Test Task"
	content := "This is a test task"
	priority := TaskPriorityMedium

	task := NewTask(name, content, priority)

	if task.Name != name {
		t.Errorf("Expected name %s, got %s", name, task.Name)
	}
	if task.Content != content {
		t.Errorf("Expected content %s, got %s", content, task.Content)
	}
	if task.Status != TaskStatusPending {
		t.Errorf("Expected status pending, got %v", task.Status)
	}
	if task.Priority != priority {
		t.Errorf("Expected priority %v, got %v", priority, task.Priority)
	}
	if task.CreatedAt.IsZero() {
		t.Errorf("Expected created at to be set")
	}
	if task.UpdatedAt.IsZero() {
		t.Errorf("Expected updated at to be set")
	}
}

func TestTask_FilterValue(t *testing.T) {
	task := &Task{Name: "Test Task"}

	if task.FilterValue() != "Test Task" {
		t.Errorf("Expected filter value 'Test Task', got %s", task.FilterValue())
	}
}

func TestTask_Title(t *testing.T) {
	task := &Task{Name: "Test Task"}

	if task.Title() != "Test Task" {
		t.Errorf("Expected title 'Test Task', got %s", task.Title())
	}
}

func TestTask_Description(t *testing.T) {
	// Test without due date
	task := &Task{
		Status: TaskStatusInProgress,
	}

	expected := "in_progress"
	result := task.Description()

	if result != expected {
		t.Errorf("Expected description %s, got %s", expected, result)
	}

	// Test with due date
	dueDate := time.Date(2023, 12, 25, 0, 0, 0, 0, time.UTC)
	task.DueDate = &dueDate

	expectedWithDue := "in_progress | Due: 2023-12-25"
	result = task.Description()

	if result != expectedWithDue {
		t.Errorf("Expected description %s, got %s", expectedWithDue, result)
	}
}

func TestNewMessage(t *testing.T) {
	role := "user"
	content := "Hello world"

	message := NewMessage(role, content)

	if message.Role != role {
		t.Errorf("Expected role %s, got %s", role, message.Role)
	}
	if message.Content != content {
		t.Errorf("Expected content %s, got %s", content, message.Content)
	}
	if message.Timestamp.IsZero() {
		t.Errorf("Expected timestamp to be set")
	}
}

func TestMessage_AddUsage(t *testing.T) {
	message := &Message{}

	promptTokens := 100
	completionTokens := 200
	inputCostPerMille := 3.00
	outputCostPerMille := 15.00

	message.AddUsage(promptTokens, completionTokens, inputCostPerMille, outputCostPerMille)

	expectedInputCost := float64(promptTokens) * inputCostPerMille / 1000000.0
	expectedOutputCost := float64(completionTokens) * outputCostPerMille / 1000000.0
	expectedTotalCost := expectedInputCost + expectedOutputCost

	if message.Usage == nil {
		t.Fatal("Expected usage to be initialized")
	}
	if message.Usage.PromptTokens != promptTokens {
		t.Errorf("Expected prompt tokens %d, got %d", promptTokens, message.Usage.PromptTokens)
	}
	if message.Usage.CompletionTokens != completionTokens {
		t.Errorf("Expected completion tokens %d, got %d", completionTokens, message.Usage.CompletionTokens)
	}
	if message.Usage.TotalTokens != promptTokens+completionTokens {
		t.Errorf("Expected total tokens %d, got %d", promptTokens+completionTokens, message.Usage.TotalTokens)
	}
	if message.Usage.Cost != expectedTotalCost {
		t.Errorf("Expected cost %f, got %f", expectedTotalCost, message.Usage.Cost)
	}
}

func TestNewToolData(t *testing.T) {
	toolType := "FileRead"
	name := "File Read Tool"
	description := "Reads files from disk"
	config := map[string]string{"workspace": "/tmp"}

	toolData := NewToolData(toolType, name, description, config)

	if toolData.ToolType != toolType {
		t.Errorf("Expected tool type %s, got %s", toolType, toolData.ToolType)
	}
	if toolData.Name != name {
		t.Errorf("Expected name %s, got %s", name, toolData.Name)
	}
	if toolData.Description != description {
		t.Errorf("Expected description %s, got %s", description, toolData.Description)
	}
	if len(toolData.Configuration) != len(config) {
		t.Errorf("Expected %d config items, got %d", len(config), len(toolData.Configuration))
	}
	if toolData.CreatedAt.IsZero() {
		t.Errorf("Expected created at to be set")
	}
	if toolData.UpdatedAt.IsZero() {
		t.Errorf("Expected updated at to be set")
	}
}

func TestToolItem_FilterValue(t *testing.T) {
	toolData := ToolData{Name: "Test Tool"}
	toolItem := ToolItem{Tool: toolData}

	if toolItem.FilterValue() != "Test Tool" {
		t.Errorf("Expected filter value 'Test Tool', got %s", toolItem.FilterValue())
	}
}

func TestToolItem_Title(t *testing.T) {
	toolData := ToolData{Name: "Test Tool"}
	toolItem := ToolItem{Tool: toolData}

	if toolItem.Title() != "Test Tool" {
		t.Errorf("Expected title 'Test Tool', got %s", toolItem.Title())
	}
}

func TestToolItem_Description(t *testing.T) {
	description := "A test tool description"
	toolData := ToolData{Description: description}
	toolItem := ToolItem{Tool: toolData}

	if toolItem.Description() != description {
		t.Errorf("Expected description %s, got %s", description, toolItem.Description())
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}
