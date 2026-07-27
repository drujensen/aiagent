package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/services"

	"go.uber.org/zap"
)

// skillDepthKey is an unexported context key used to thread the
// skill-invocation depth counter through the recursive call chain:
// SkillTool.Execute -> ChatService.ExecuteSkill -> ChatService.SendMessage
// -> (the target chat's LLM decides to call SkillTool again) ->
// SkillTool.Execute. Using context.Value rather than a tool argument means
// the depth cannot be spoofed, omitted, or forgotten by the calling LLM -
// ctx flows unmodified through this entire call chain (SendMessage passes
// the same ctx into GenerateResponse, which passes it into every tool's
// Execute), so a value attached here is visible however many levels deep
// the recursion goes.
type skillDepthKey struct{}

// maxSkillDepth bounds nested skill invocations (a skill running in chat A
// invokes a skill in chat B, whose skill invokes a skill in chat C, ...) to
// prevent unbounded recursion.
const maxSkillDepth = 3

// maxAncestorWalk bounds the ancestor-chain walk used by the self/ancestor
// targeting guard, defensively, in case a chat's ParentChatID chain is
// unexpectedly long.
const maxAncestorWalk = 64

func skillDepthFromContext(ctx context.Context) int {
	if d, ok := ctx.Value(skillDepthKey{}).(int); ok {
		return d
	}
	return 0
}

func withSkillDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, skillDepthKey{}, depth)
}

// SkillTool invokes a named pipeline skill against a target chat by
// injecting the skill's instructions as a message into that chat (via
// ChatService.ExecuteSkill). Services are accessed lazily through the
// ToolFactory, matching AgentTool's pattern, so this tool can be
// instantiated before services are wired up in main.
type SkillTool struct {
	name          string
	description   string
	configuration map[string]string
	factory       *ToolFactory
	logger        *zap.Logger
}

func NewSkillTool(name, description string, configuration map[string]string, factory *ToolFactory, logger *zap.Logger) *SkillTool {
	return &SkillTool{
		name:          name,
		description:   description,
		configuration: configuration,
		factory:       factory,
		logger:        logger,
	}
}

func (t *SkillTool) Name() string { return t.name }

func (t *SkillTool) Description() string { return t.description }

func (t *SkillTool) Configuration() map[string]string { return t.configuration }

func (t *SkillTool) UpdateConfiguration(config map[string]string) { t.configuration = config }

func (t *SkillTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- chat_id: The target chat to invoke the skill against. Must not be the invoking chat itself or any of its ancestors.\n- skill_name: The name of the skill to invoke.", t.Description())
}

func (t *SkillTool) Schema() map[string]any {
	skillNameDesc := "Name of the skill to invoke."

	// Schema() is called at request time, after services are wired, so it
	// is safe to query the skill service here (same pattern as AgentTool's
	// dynamic agent_name enumeration).
	if svc := t.factory.GetSkillService(); svc != nil {
		if skills, err := svc.ListSkills(context.Background()); err == nil && len(skills) > 0 {
			names := make([]string, len(skills))
			for i, s := range skills {
				names[i] = s.Name
			}
			skillNameDesc = "Name of the skill to invoke. Available skills: " + strings.Join(names, ", ") + "."
		}
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"chat_id": map[string]any{
				"type":        "string",
				"description": "The target chat to invoke the skill against.",
			},
			"skill_name": map[string]any{
				"type":        "string",
				"description": skillNameDesc,
			},
		},
		"required":             []string{"chat_id", "skill_name"},
		"additionalProperties": false,
	}
}

func (t *SkillTool) Execute(ctx context.Context, arguments string) (string, error) {
	chatService := t.factory.GetChatService()
	if chatService == nil {
		return "", fmt.Errorf("skill tool not ready: services not yet initialized")
	}

	var args struct {
		ChatID       string `json:"chat_id"`
		SkillName    string `json:"skill_name"`
		ParentChatID string `json:"parent_chat_id"` // injected by the framework via injectToolArgs: the invoking chat's own ID
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}
	if args.ChatID == "" {
		return "", fmt.Errorf("chat_id is required")
	}
	if args.SkillName == "" {
		return "", fmt.Errorf("skill_name is required")
	}

	depth := skillDepthFromContext(ctx)
	if depth >= maxSkillDepth {
		return "", fmt.Errorf("maximum skill-invocation depth (%d) exceeded", maxSkillDepth)
	}

	invokingChatID := args.ParentChatID
	if invokingChatID != "" {
		if args.ChatID == invokingChatID {
			return "", fmt.Errorf("skill target chat_id %q must not equal the invoking chat's own ID", args.ChatID)
		}
		if err := rejectIfAncestor(ctx, chatService, invokingChatID, args.ChatID); err != nil {
			return "", err
		}
	}

	if err := chatService.ExecuteSkill(withSkillDepth(ctx, depth+1), args.ChatID, args.SkillName); err != nil {
		return "", fmt.Errorf("failed to execute skill %q against chat %s: %w", args.SkillName, args.ChatID, err)
	}

	t.logger.Info("Skill executed", zap.String("skill", args.SkillName), zap.String("target_chat_id", args.ChatID))
	return fmt.Sprintf("Skill %q executed against chat %s.", args.SkillName, args.ChatID), nil
}

// rejectIfAncestor walks invokingChatID's ParentChatID chain and returns an
// error if targetChatID appears in it. This closes a two-chat recursion gap
// that a single self-equality check would miss: chat A invokes a skill
// targeting chat B (a descendant of A, e.g. created via AgentTool/
// PlanService fan-out); B's agent then invokes a skill targeting A. At that
// second invocation the invoking chat is B and the target is A - not equal
// to B, but an ancestor of B, and therefore still a cycle back to A.
func rejectIfAncestor(ctx context.Context, chatService services.ChatService, invokingChatID, targetChatID string) error {
	current := invokingChatID
	for i := 0; i < maxAncestorWalk; i++ {
		chat, err := chatService.GetChat(ctx, current)
		if err != nil || chat.ParentChatID == "" {
			return nil
		}
		if chat.ParentChatID == targetChatID {
			return fmt.Errorf("skill target chat_id %q is an ancestor of the invoking chat; rejected to prevent unbounded skill recursion", targetChatID)
		}
		current = chat.ParentChatID
	}
	return nil
}

func (t *SkillTool) DisplayName(ui string, arguments string) (string, string) {
	var args struct {
		SkillName string `json:"skill_name"`
	}
	json.Unmarshal([]byte(arguments), &args) //nolint:errcheck
	return t.Name(), args.SkillName
}

func (t *SkillTool) FormatResult(ui string, result string, diff string, arguments string) string {
	return result
}

var _ entities.Tool = (*SkillTool)(nil)
