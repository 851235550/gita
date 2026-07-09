package llm

import (
	"context"
	"fmt"
)

// MockProvider 是 Provider 接口的 mock 实现，用于单元测试。
// 支持预设返回值、模拟错误、记录调用次数等场景，
// 避免单元测试中发起真实网络请求（开发规范 1.3 节）。
type MockProvider struct {
	// GenerateFunc 为可自定义的 Generate 行为，若为 nil 则使用默认行为（返回预设文本）。
	GenerateFunc func(ctx context.Context, prompt string) (string, error)

	// CallCount 记录 Generate 被调用的次数，每次调用自动 +1。
	CallCount int

	// LastPrompt 保存最后一次调用时传入的 prompt，用于断言 prompt 内容是否符合预期。
	LastPrompt string
}

// Generate 实现 Provider 接口。
// 若 GenerateFunc 不为 nil，委托给 GenerateFunc；否则返回默认 mock 文本。
// 每次调用将 CallCount +1 并记录 LastPrompt。
func (m *MockProvider) Generate(ctx context.Context, prompt string) (string, error) {
	m.CallCount++
	m.LastPrompt = prompt

	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt)
	}

	// 默认行为：返回固定的 commit message，供交互流程测试使用。
	return "feat(mock): mock generated commit message", nil
}

// NewErrorMock 创建一个始终返回指定错误的 MockProvider，用于模拟 LLM 调用失败场景。
func NewErrorMock(errMsg string) *MockProvider {
	return &MockProvider{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", fmt.Errorf("%s", errMsg)
		},
	}
}

// NewEmptyMock 创建一个始终返回空字符串的 MockProvider，用于模拟 LLM 返回空内容场景。
func NewEmptyMock() *MockProvider {
	return &MockProvider{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", nil
		},
	}
}
