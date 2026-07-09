package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"gita/internal/config"
)

// testProviderConfig 返回用于测试的 DeepSeek Provider 配置。
func testProviderConfig() *config.ProviderConfig {
	return &config.ProviderConfig{
		Model:     "deepseek-chat",
		BaseURL:   "https://api.deepseek.com/v1",
		APIKeyEnv: "GITA_DEEPSEEK_API_KEY",
	}
}

// ============================================================================
// TC-LLM-01: 未设置 API Key 环境变量 → 返回错误，含环境变量名
// ============================================================================
func TestNewDeepSeekProvider_NoAPIKey_ReturnsError(t *testing.T) {
	// Arrange: 确保环境变量未设置。
	os.Unsetenv("GITA_DEEPSEEK_API_KEY")

	// Act
	_, err := NewDeepSeekProvider(testProviderConfig(), "")

	// Assert
	if err == nil {
		t.Fatal("未设置 API Key 时应返回错误")
	}
	if !strings.Contains(err.Error(), "GITA_DEEPSEEK_API_KEY") {
		t.Errorf("错误信息应包含环境变量名 GITA_DEEPSEEK_API_KEY，实际: %v", err)
	}
}

// ============================================================================
// TC-LLM-02: API Key 无效 → 返回错误（通过 mock server 返回 401）
// ============================================================================
func TestDeepSeekProvider_InvalidAPIKey_ReturnsError(t *testing.T) {
	// Arrange: mock server 返回 401。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "invalid-key",
		model:   "deepseek-chat",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Act
	_, err := provider.Generate(context.Background(), "test prompt")

	// Assert
	if err == nil {
		t.Fatal("无效 API Key 应返回错误")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("错误信息应包含 HTTP 状态码 401，实际: %v", err)
	}
}

// ============================================================================
// TC-LLM-03: Mock server 模拟 35 秒无响应 → 30 秒超时
// ============================================================================
func TestDeepSeekProvider_Timeout_ReturnsError(t *testing.T) {
	// Arrange: mock server 延迟 2 秒响应，但 client 超时设为 50ms。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "test-key",
		model:   "deepseek-chat",
		baseURL: server.URL,
		httpClient: &http.Client{
			// 使用极短超时模拟 30s 超时场景（避免测试真的等 30s）。
			Timeout: 50 * time.Millisecond,
		},
	}

	// Act
	_, err := provider.Generate(context.Background(), "test prompt")

	// Assert
	if err == nil {
		t.Fatal("超时场景应返回错误")
	}
	// 错误信息应包含"超时"关键字。
	if !strings.Contains(err.Error(), "超时") {
		t.Errorf("超时错误信息应包含'超时'，实际: %v", err)
	}
}

// ============================================================================
// TC-LLM-04: Mock server 返回空 content → 返回"内容为空"错误
// ============================================================================
func TestDeepSeekProvider_EmptyContent_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":""}}]}`))
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "test-key",
		model:   "deepseek-chat",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Act
	_, err := provider.Generate(context.Background(), "test prompt")

	// Assert
	if err == nil {
		t.Fatal("空 content 应返回错误")
	}
	if !strings.Contains(err.Error(), "内容为空") {
		t.Errorf("错误信息应包含'内容为空'，实际: %v", err)
	}
}

// ============================================================================
// TC-LLM-05: Mock server 返回格式错误的 JSON → 返回解析错误
// ============================================================================
func TestDeepSeekProvider_MalformedJSON_ReturnsError(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`this is not json`))
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "test-key",
		model:   "deepseek-chat",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Act
	_, err := provider.Generate(context.Background(), "test prompt")

	// Assert
	if err == nil {
		t.Fatal("格式错误 JSON 应返回错误")
	}
}

// ============================================================================
// TC-LLM-06: 正常场景，Mock server 返回合法结果
// ============================================================================
func TestDeepSeekProvider_NormalResponse_ReturnsContent(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"feat(test): add unit test"}}]}`))
	}))
	defer server.Close()

	provider := &DeepSeekProvider{
		apiKey:  "test-key",
		model:   "deepseek-chat",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Act
	result, err := provider.Generate(context.Background(), "test prompt")

	// Assert
	if err != nil {
		t.Fatalf("正常响应不应报错: %v", err)
	}
	expected := "feat(test): add unit test"
	if result != expected {
		t.Errorf("result = %q, want %q", result, expected)
	}
}

// ============================================================================
// 补充: --api-key 参数优先级高于环境变量
// ============================================================================
func TestNewDeepSeekProvider_APIKeyOverride_Priority(t *testing.T) {
	// Arrange: 设置环境变量，但同时传入 --api-key 覆盖值。
	t.Setenv("GITA_DEEPSEEK_API_KEY", "env-key")

	// Act: 使用 override 参数传入临时 Key。
	provider, err := NewDeepSeekProvider(testProviderConfig(), "override-key")

	// Assert
	if err != nil {
		t.Fatalf("创建 Provider 失败: %v", err)
	}
	// 应使用 override 的 Key 而非环境变量中的 Key。
	if provider.apiKey != "override-key" {
		t.Errorf("apiKey = %q, want %q（--api-key 应优先于环境变量）", provider.apiKey, "override-key")
	}
}

// ============================================================================
// 补充: 网络不可达（mock 关闭的 server）
// ============================================================================
func TestDeepSeekProvider_NetworkError_ReturnsError(t *testing.T) {
	// Arrange: 使用一个已关闭的 server 地址。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // 立即关闭，模拟网络不可达。

	provider := &DeepSeekProvider{
		apiKey:  "test-key",
		model:   "deepseek-chat",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 1 * time.Second,
		},
	}

	// Act
	_, err := provider.Generate(context.Background(), "test prompt")

	// Assert
	if err == nil {
		t.Fatal("网络不可达应返回错误")
	}
}
