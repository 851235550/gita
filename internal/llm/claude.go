package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gita/internal/config"
)

// ClaudeProvider 实现 Provider 接口，对接 Anthropic Claude Messages API。
// API 文档: https://docs.anthropic.com/en/api/messages
// Claude 使用与 OpenAI 不同的请求/响应格式，需单独适配。
type ClaudeProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// claudeRequest 是 Anthropic Messages API 的请求体。
// 与 OpenAI Chat Completions 格式不同，Claude 使用独立的 messages/content 结构。
type claudeRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []claudeMessage  `json:"messages"`
}

// claudeMessage 是 Claude API 的单条消息。
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse 是 Anthropic Messages API 的响应体。
// Claude 的响应格式为 content 数组，每个元素含 type 和 text 字段。
type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// NewClaudeProvider 根据配置创建 Claude Provider。
func NewClaudeProvider(cfg *config.ProviderConfig, apiKeyOverride string) (*ClaudeProvider, error) {
	apiKey := apiKeyOverride
	if apiKey == "" {
		apiKey = os.Getenv(cfg.APIKeyEnv)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("未找到 Anthropic API Key，请设置环境变量 %s 或使用 --api-key 参数", cfg.APIKeyEnv)
	}

	return &ClaudeProvider{
		apiKey:  apiKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Generate 向 Anthropic Messages API 发送 prompt 并返回生成的文本。
func (p *ClaudeProvider) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     p.model,
		MaxTokens: 4096, // 足够覆盖绝大多数 commit message 的长度需求。
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("构造 Claude 请求失败: %w", err)
	}

	// Anthropic Messages API 端点路径不同于 OpenAI。
	url := p.baseURL + "/messages"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Anthropic 使用 x-api-key header，而非 Authorization: Bearer。
	req.Header.Set("x-api-key", p.apiKey)
	// Anthropic 要求的 API 版本头。
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") ||
			strings.Contains(errStr, "Timeout") {
			return "", fmt.Errorf("LLM 请求超时（超过 30 秒），请检查网络或稍后重试: %w", err)
		}
		return "", fmt.Errorf("LLM 请求失败（网络不可达）: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Claude 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API 返回错误状态 %d: %s", resp.StatusCode, string(respBytes))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBytes, &claudeResp); err != nil {
		return "", fmt.Errorf("解析 Claude 响应失败（JSON 格式异常）: %w", err)
	}

	// Claude 响应中 content 是数组，取第一个 text 类型的内容。
	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("LLM 返回内容为空，可尝试重新生成")
	}
	// 提取所有 text 类型的 content 并拼接（通常只有一个）。
	var texts []string
	for _, c := range claudeResp.Content {
		if c.Type == "text" && c.Text != "" {
			texts = append(texts, c.Text)
		}
	}
	result := strings.Join(texts, "\n")
	if result == "" {
		return "", fmt.Errorf("LLM 返回内容为空，可尝试重新生成")
	}

	return result, nil
}
