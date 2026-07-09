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

// OpenAIProvider 实现 Provider 接口，对接 OpenAI Chat Completions API。
// API 文档: https://platform.openai.com/docs/api-reference/chat
type OpenAIProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider 根据配置创建 OpenAI Provider。
// cfg 中的 api_key_env 指定了存放 Key 的环境变量名。
// apiKeyOverride 为命令行 --api-key 传入的临时 Key，优先级最高。
func NewOpenAIProvider(cfg *config.ProviderConfig, apiKeyOverride string) (*OpenAIProvider, error) {
	apiKey := apiKeyOverride
	if apiKey == "" {
		apiKey = os.Getenv(cfg.APIKeyEnv)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("未找到 OpenAI API Key，请设置环境变量 %s 或使用 --api-key 参数", cfg.APIKeyEnv)
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Generate 向 OpenAI API 发送 prompt 并返回生成的文本。
// OpenAI API 与 DeepSeek 使用兼容的 Chat Completions 格式。
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("构造 OpenAI 请求失败: %w", err)
	}

	url := p.baseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		return "", fmt.Errorf("读取 OpenAI 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API 返回错误状态 %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("解析 OpenAI 响应失败（JSON 格式异常）: %w", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("LLM 返回内容为空，可尝试重新生成")
	}

	return chatResp.Choices[0].Message.Content, nil
}
