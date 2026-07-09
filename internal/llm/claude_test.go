package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gita/internal/config"
)

func claudeTestConfig() *config.ProviderConfig {
	return &config.ProviderConfig{
		Model:     "claude-sonnet-4-6",
		BaseURL:   "https://api.anthropic.com/v1",
		APIKeyEnv: "GITA_ANTHROPIC_API_KEY",
	}
}

// ============================================================================
// TC-LLM-01: 未设置 API Key
// ============================================================================
func TestNewClaudeProvider_NoAPIKey_ReturnsError(t *testing.T) {
	// Arrange
	t.Setenv("GITA_ANTHROPIC_API_KEY", "")

	// Act
	_, err := NewClaudeProvider(claudeTestConfig(), "")

	// Assert
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !strings.Contains(err.Error(), "GITA_ANTHROPIC_API_KEY") {
		t.Errorf("错误应包含环境变量名: %v", err)
	}
}

// ============================================================================
// TC-LLM-02: API Key 无效 → 401
// ============================================================================
func TestClaudeProvider_InvalidAPIKey_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 Anthropic 特有的请求头。
		if r.Header.Get("x-api-key") == "" {
			t.Error("Claude 请求应包含 x-api-key 头")
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "invalid x-api-key"}}`))
	}))
	defer server.Close()

	provider := &ClaudeProvider{
		apiKey:  "bad-key",
		model:   "claude-sonnet-4-6",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Act
	_, err := provider.Generate(context.Background(), "prompt")

	// Assert
	if err == nil {
		t.Fatal("应返回错误")
	}
}

// ============================================================================
// TC-LLM-06: 正常场景 —— Claude 特有响应格式
// ============================================================================
func TestClaudeProvider_NormalResponse_ReturnsContent(t *testing.T) {
	// Arrange: Claude 的 content 是数组格式，不同于 OpenAI。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"content":[{"type":"text","text":"feat(claude): add claude integration"}]}`))
	}))
	defer server.Close()

	provider := &ClaudeProvider{
		apiKey:  "test-key",
		model:   "claude-sonnet-4-6",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Act
	result, err := provider.Generate(context.Background(), "prompt")

	// Assert
	if err != nil {
		t.Fatalf("正常响应不应报错: %v", err)
	}
	if result != "feat(claude): add claude integration" {
		t.Errorf("result = %q", result)
	}
}

// ============================================================================
// TC-LLM-04: 空 content 数组
// ============================================================================
func TestClaudeProvider_EmptyContent_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	provider := &ClaudeProvider{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-6",
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	// Act
	_, err := provider.Generate(context.Background(), "prompt")

	// Assert
	if err == nil {
		t.Fatal("空 content 应返回错误")
	}
}

// ============================================================================
// TC-LLM-05: 格式错误 JSON
// ============================================================================
func TestClaudeProvider_MalformedJSON_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	provider := &ClaudeProvider{
		apiKey:     "test-key",
		model:      "claude-sonnet-4-6",
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	// Act
	_, err := provider.Generate(context.Background(), "prompt")

	// Assert
	if err == nil {
		t.Fatal("格式错误 JSON 应返回错误")
	}
}
