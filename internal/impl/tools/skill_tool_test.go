package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// mockChatServiceForSkill is a from-scratch mock of the 13-method
// ChatService interface. A separate mock from plan_service_test.go's
// mockChatServiceForPlan is required here: that mock lives in package
// services and isn't importable from this package's test file, and per the
// plan this is genuinely new test infrastructure, not reuse of an existing
// mock.
type mockChatServiceForSkill struct {
	mock.Mock
}

func (m *mockChatServiceForSkill) ListChats(ctx context.Context) ([]*entities.Chat, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entities.Chat), args.Error(1)
}

func (m *mockChatServiceForSkill) GetChat(ctx context.Context, id string) (*entities.Chat, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForSkill) GetActiveChat(ctx context.Context) (*entities.Chat, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForSkill) SetActiveChat(ctx context.Context, chatID string) error {
	args := m.Called(ctx, chatID)
	return args.Error(0)
}

func (m *mockChatServiceForSkill) CreateChat(ctx context.Context, agentID, modelID, name string) (*entities.Chat, error) {
	args := m.Called(ctx, agentID, modelID, name)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForSkill) CreateSubChat(ctx context.Context, agentID, modelID, name, parentChatID string) (*entities.Chat, error) {
	args := m.Called(ctx, agentID, modelID, name, parentChatID)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForSkill) UpdateChat(ctx context.Context, id, agentID, modelID, name string) (*entities.Chat, error) {
	args := m.Called(ctx, id, agentID, modelID, name)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForSkill) DeleteChat(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockChatServiceForSkill) SendMessage(ctx context.Context, id string, message *entities.Message) (*entities.Message, error) {
	args := m.Called(ctx, id, message)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Message), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForSkill) SaveMessagesIncrementally(ctx context.Context, chatID string, messages []*entities.Message) error {
	args := m.Called(ctx, chatID, messages)
	return args.Error(0)
}

func (m *mockChatServiceForSkill) CalculateTotalChatCost(ctx context.Context, chatID string) (float64, error) {
	args := m.Called(ctx, chatID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockChatServiceForSkill) GenerateAndUpdateTitle(ctx context.Context, chatID string) (*entities.Chat, error) {
	args := m.Called(ctx, chatID)
	if args.Get(0) != nil {
		return args.Get(0).(*entities.Chat), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatServiceForSkill) ExecuteSkill(ctx context.Context, chatID, skillName string) error {
	args := m.Called(ctx, chatID, skillName)
	return args.Error(0)
}

var _ services.ChatService = (*mockChatServiceForSkill)(nil)

func newSkillToolWithMockChat(mockChat *mockChatServiceForSkill) *SkillTool {
	factory := &ToolFactory{}
	factory.SetServices(mockChat, nil, nil)
	return NewSkillTool("Skill", "test", map[string]string{}, factory, zap.NewNop())
}

// TestSkillTool_InvokesSkillAgainstNonActiveChat is AC6a's deliverable
// test: SkillTool invokes ExecuteSkill against a target chat that is not
// the invoking chat, verified with a mocked ChatService.
func TestSkillTool_InvokesSkillAgainstNonActiveChat(t *testing.T) {
	mockChat := new(mockChatServiceForSkill)
	tool := newSkillToolWithMockChat(mockChat)

	invokingChat := &entities.Chat{ID: "chat-A", ParentChatID: ""}
	mockChat.On("GetChat", mock.Anything, "chat-A").Return(invokingChat, nil)
	mockChat.On("ExecuteSkill", mock.Anything, "chat-B", "research").Return(nil)

	argsJSON, err := json.Marshal(map[string]any{
		"chat_id":        "chat-B",
		"skill_name":     "research",
		"parent_chat_id": "chat-A",
	})
	assert.NoError(t, err)

	result, err := tool.Execute(context.Background(), string(argsJSON))
	assert.NoError(t, err)
	assert.Contains(t, result, "chat-B")
	mockChat.AssertCalled(t, "ExecuteSkill", mock.Anything, "chat-B", "research")
}

// TestSkillTool_RejectsSelfTargeting is part of AC6d.
func TestSkillTool_RejectsSelfTargeting(t *testing.T) {
	mockChat := new(mockChatServiceForSkill)
	tool := newSkillToolWithMockChat(mockChat)

	argsJSON, err := json.Marshal(map[string]any{
		"chat_id":        "chat-A",
		"skill_name":     "research",
		"parent_chat_id": "chat-A",
	})
	assert.NoError(t, err)

	_, err = tool.Execute(context.Background(), string(argsJSON))
	assert.Error(t, err)
	mockChat.AssertNotCalled(t, "ExecuteSkill", mock.Anything, mock.Anything, mock.Anything)
}

// TestSkillTool_RejectsTwoChatCycle is AC6d's two-chat recursion test: chat
// A invokes a skill targeting chat B (a descendant of A - allowed), then
// B's agent invokes a skill targeting A. The second invocation must be
// rejected because A is an ancestor of the invoking chat B, even though A
// is not equal to B.
func TestSkillTool_RejectsTwoChatCycle(t *testing.T) {
	mockChat := new(mockChatServiceForSkill)
	tool := newSkillToolWithMockChat(mockChat)

	chatA := &entities.Chat{ID: "A", ParentChatID: ""}
	chatB := &entities.Chat{ID: "B", ParentChatID: "A"}
	mockChat.On("GetChat", mock.Anything, "A").Return(chatA, nil)
	mockChat.On("GetChat", mock.Anything, "B").Return(chatB, nil)
	mockChat.On("ExecuteSkill", mock.Anything, "B", "design").Return(nil)

	// Hop 1: A invokes a skill targeting B - allowed (B is a descendant of A,
	// not an ancestor of A).
	hop1Args, err := json.Marshal(map[string]any{
		"chat_id":        "B",
		"skill_name":     "design",
		"parent_chat_id": "A",
	})
	assert.NoError(t, err)
	_, err = tool.Execute(context.Background(), string(hop1Args))
	assert.NoError(t, err, "A invoking a skill targeting its own descendant B should be allowed")

	// Hop 2: B invokes a skill targeting A - rejected (A is an ancestor of
	// the invoking chat B).
	hop2Args, err := json.Marshal(map[string]any{
		"chat_id":        "A",
		"skill_name":     "plan",
		"parent_chat_id": "B",
	})
	assert.NoError(t, err)
	_, err = tool.Execute(context.Background(), string(hop2Args))
	assert.Error(t, err, "B invoking a skill targeting its own ancestor A must be rejected")
	mockChat.AssertNotCalled(t, "ExecuteSkill", mock.Anything, "A", "plan")
}

// TestSkillTool_RejectsDepthFourChain is AC6d's depth test: a chain of 4
// nested skill invocations (depths 0, 1, 2, 3) must reject the 4th (depth
// 3, since maxSkillDepth is 3 and the guard is depth >= maxSkillDepth).
// The depth counter is threaded via context, mirroring how it would
// actually propagate through nested ChatService.SendMessage calls.
func TestSkillTool_RejectsDepthFourChain(t *testing.T) {
	mockChat := new(mockChatServiceForSkill)
	tool := newSkillToolWithMockChat(mockChat)
	mockChat.On("ExecuteSkill", mock.Anything, "target", "research").Return(nil)

	argsJSON, err := json.Marshal(map[string]any{
		"chat_id":    "target",
		"skill_name": "research",
	})
	assert.NoError(t, err)

	for depth := 0; depth < 3; depth++ {
		ctx := withSkillDepth(context.Background(), depth)
		_, err := tool.Execute(ctx, string(argsJSON))
		assert.NoError(t, err, "depth %d should be allowed (below maxSkillDepth)", depth)
	}

	ctx := withSkillDepth(context.Background(), 3)
	_, err = tool.Execute(ctx, string(argsJSON))
	assert.Error(t, err, "depth 3 (the 4th invocation in the chain) must be rejected")
}

func TestSkillTool_RequiresChatIDAndSkillName(t *testing.T) {
	mockChat := new(mockChatServiceForSkill)
	tool := newSkillToolWithMockChat(mockChat)

	cases := []map[string]any{
		{"skill_name": "research"},
		{"chat_id": "chat-B"},
	}
	for _, args := range cases {
		argsJSON, err := json.Marshal(args)
		assert.NoError(t, err)
		_, err = tool.Execute(context.Background(), string(argsJSON))
		assert.Error(t, err)
	}
}
