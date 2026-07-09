package llm

import (
	"context"
	"strings"
	"testing"
)

// ============================================================================
// MockProvider 基本行为测试
// ============================================================================

func TestMockProvider_DefaultBehavior(t *testing.T) {
	// Arrange
	mock := &MockProvider{}
	ctx := context.Background()

	// Act
	result, err := mock.Generate(ctx, "test prompt")

	// Assert
	if err != nil {
		t.Fatalf("默认 MockProvider 不应返回错误: %v", err)
	}
	if result == "" {
		t.Error("默认 MockProvider 应返回非空文本")
	}
	if mock.CallCount != 1 {
		t.Errorf("CallCount = %d, want 1", mock.CallCount)
	}
	if mock.LastPrompt != "test prompt" {
		t.Errorf("LastPrompt = %q, want %q", mock.LastPrompt, "test prompt")
	}
}

func TestMockProvider_CallCount_Increments(t *testing.T) {
	// Arrange
	mock := &MockProvider{}
	ctx := context.Background()

	// Act: 连续调用 3 次。
	for i := 0; i < 3; i++ {
		_, _ = mock.Generate(ctx, "prompt")
	}

	// Assert
	if mock.CallCount != 3 {
		t.Errorf("CallCount = %d, want 3", mock.CallCount)
	}
}

func TestMockProvider_CustomGenerateFunc(t *testing.T) {
	// Arrange: 自定义 GenerateFunc 返回特定文本。
	mock := &MockProvider{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "custom response for: " + prompt, nil
		},
	}
	ctx := context.Background()

	// Act
	result, err := mock.Generate(ctx, "hello")

	// Assert
	if err != nil {
		t.Fatalf("自定义 GenerateFunc 不应返回错误: %v", err)
	}
	if result != "custom response for: hello" {
		t.Errorf("result = %q, want %q", result, "custom response for: hello")
	}
}

func TestNewErrorMock_ReturnsError(t *testing.T) {
	// Arrange
	mock := NewErrorMock("LLM timeout")
	ctx := context.Background()

	// Act
	_, err := mock.Generate(ctx, "prompt")

	// Assert
	if err == nil {
		t.Fatal("ErrorMock 应返回错误")
	}
	if !strings.Contains(err.Error(), "LLM timeout") {
		t.Errorf("错误信息应包含 'LLM timeout'，实际: %v", err)
	}
}

func TestNewEmptyMock_ReturnsEmptyString(t *testing.T) {
	// Arrange
	mock := NewEmptyMock()
	ctx := context.Background()

	// Act
	result, err := mock.Generate(ctx, "prompt")

	// Assert
	if err != nil {
		t.Fatalf("EmptyMock 不应返回错误: %v", err)
	}
	if result != "" {
		t.Errorf("EmptyMock 应返回空字符串，实际: %q", result)
	}
}
