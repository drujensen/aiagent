package services

import (
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestTrimMessagesToLimit(t *testing.T) {
	logger := zap.NewNop()

	// Create a mock chatService (we only need the logger for trimMessagesToLimit)
	cs := &chatService{
		logger: logger,
	}

	// Create test messages
	messages := []*entities.Message{
		{ID: uuid.New().String(), Role: "system", Content: "You are a helpful assistant"},
		{ID: uuid.New().String(), Role: "user", Content: "Hello"},
		{ID: uuid.New().String(), Role: "assistant", Content: "Hi there!"},
		{ID: uuid.New().String(), Role: "user", Content: "How are you?"},
		{ID: uuid.New().String(), Role: "assistant", Content: "I'm doing well, thank you!"},
	}

	// Test with a very low token limit to force trimming
	result := cs.trimMessagesToLimit(messages, 10) // Very low limit

	// Should always keep at least the system message
	if len(result) == 0 {
		t.Error("trimMessagesToLimit should always return at least one message")
	}

	// System message should always be first
	if result[0].Role != "system" {
		t.Error("System message should always be first")
	}

	// All messages should be non-nil
	for i, msg := range result {
		if msg == nil {
			t.Errorf("Message at index %d is nil", i)
		}
	}
}

func TestTrimMessagesToLimitNilInput(t *testing.T) {
	logger := zap.NewNop()
	cs := &chatService{logger: logger}

	// Test with nil input
	result := cs.trimMessagesToLimit(nil, 100)
	if result != nil {
		t.Error("trimMessagesToLimit should return nil for nil input")
	}

	// Test with empty slice
	empty := []*entities.Message{}
	result = cs.trimMessagesToLimit(empty, 100)
	if len(result) != 0 {
		t.Error("trimMessagesToLimit should return empty slice for empty input")
	}
}

func TestEstimateTokens(t *testing.T) {
	// Test with nil message
	tokens := estimateTokens(nil)
	if tokens != 0 {
		t.Errorf("estimateTokens(nil) should return 0, got %d", tokens)
	}

	// Test with valid message
	msg := &entities.Message{
		Role:    "user",
		Content: "Hello world",
	}
	tokens = estimateTokens(msg)
	if tokens <= 0 {
		t.Error("estimateTokens should return positive token count for valid message")
	}
}

func TestIsContextError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "ContextWindowError",
			err:      errors.ContextWindowErrorf("test context error"),
			expected: true,
		},
		{
			name:     "string containing context",
			err:      errors.InternalErrorf("context window exceeded"),
			expected: true,
		},
		{
			name:     "string containing token limit",
			err:      errors.InternalErrorf("token limit reached"),
			expected: true,
		},
		{
			name:     "string containing maximum length",
			err:      errors.InternalErrorf("maximum length exceeded"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.InternalErrorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isContextError(tt.err)
			if result != tt.expected {
				t.Errorf("isContextError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}
