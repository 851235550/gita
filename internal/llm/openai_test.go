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

func openAITestConfig() *config.ProviderConfig {
	return &config.ProviderConfig{
		Model:     "gpt-4o",
		BaseURL:   "https://api.openai.com/v1",
		APIKeyEnv: "GITA_OPENAI_API_KEY",
	}
}

// ============================================================================
// TC-LLM-01: 未设置 API Key → 含环境变量名的错误
// ============================================================================
func TestNewOpenAIProvider_NoAPIKey_ReturnsError(t *testing.T) {
	// Arrange
	t.Setenv("GITA_OPENAI_API_KEY", "")

	// Act
	_, err := NewOpenAIProvider(openAITestConfig(), "")

	// Assert
	if err == nil {
		t.Fatal("应返回错误")
	}
	if !strings.Contains(err.Error(), "GITA_OPENAI_API_KEY") {
		t.Errorf("错误应包含环境变量名: %v", err)
	}
}

// ============================================================================
// TC-LLM-02: API Key 无效 → 401 错误
// ============================================================================
func TestOpenAIProvider_InvalidAPIKey_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "invalid api key"}}`))
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		apiKey:  "bad-key",
		model:   "gpt-4o",
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
// TC-LLM-06: 正常场景
// ============================================================================
func TestOpenAIProvider_NormalResponse_ReturnsContent(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"feat(api): add openai support"}}]}`))
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		apiKey:  "test-key",
		model:   "gpt-4o",
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
	if result != "feat(api): add openai support" {
		t.Errorf("result = %q", result)
	}
}

// ============================================================================
// TC-LLM-04: 空 content
// ============================================================================
func TestOpenAIProvider_EmptyContent_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
	}))
	defer server.Close()

	provider := &OpenAIProvider{
		apiKey:     "test-key",
		model:      "gpt-4o",
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
