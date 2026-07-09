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

// DeepSeekProvider 实现 Provider 接口，对接 DeepSeek Chat API。
// DeepSeek API 兼容 OpenAI Chat Completions 格式，
// 文档: https://platform.deepseek.com/api-docs
type DeepSeekProvider struct {
	// apiKey 从环境变量读取的 API Key，不在代码中硬编码或持久化。
	apiKey string

	// model 模型名称（如 deepseek-chat），由配置文件指定。
	model string

	// baseURL API 端点地址（如 https://api.deepseek.com/v1），由配置文件指定。
	baseURL string

	// httpClient 可注入的 HTTP 客户端，便于测试时替换为 mock transport。
	httpClient *http.Client
}

// NewDeepSeekProvider 根据配置创建 DeepSeek Provider。
// cfg 中的 api_key_env 指定了存放 Key 的环境变量名，
// 若该环境变量未设置则返回错误（需求文档 7.3 节 Key 优先级规则）。
// apiKeyOverride 为命令行 --api-key 传入的临时 Key，优先级最高，非空时忽略环境变量。
func NewDeepSeekProvider(cfg *config.ProviderConfig, apiKeyOverride string) (*DeepSeekProvider, error) {
	apiKey := apiKeyOverride
	if apiKey == "" {
		apiKey = os.Getenv(cfg.APIKeyEnv)
	}
	if apiKey == "" {
		// 错误信息必须包含具体缺失的环境变量名，便于用户定位。
		return nil, fmt.Errorf("未找到 DeepSeek API Key，请设置环境变量 %s 或使用 --api-key 参数", cfg.APIKeyEnv)
	}

	return &DeepSeekProvider{
		apiKey:  apiKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		// 设置 30s 超时，对应需求文档 8 节超时阈值。
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// chatRequest 是 OpenAI 兼容的 Chat Completions 请求体。
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage 是单条对话消息。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse 是 OpenAI 兼容的 Chat Completions 响应体，仅提取需要的字段。
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Generate 向 DeepSeek API 发送 prompt 并返回生成的文本。
// ctx 用于超时控制，prompt 为已渲染完成的最终提示文本。
func (p *DeepSeekProvider) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := chatRequest{
		Model: p.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("构造 DeepSeek 请求失败: %w", err)
	}

	// DeepSeek API 的 Chat Completions 端点路径。
	url := p.baseURL + "/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// 区分超时错误与网络不可达错误，便于上层给出不同提示。
		// http.Client.Timeout 触发时，底层 context 可能不是调用方传入的 ctx，
		// 因此通过错误信息关键字而非 ctx.Err() 来判断超时。
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
		return "", fmt.Errorf("读取 DeepSeek 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DeepSeek API 返回错误状态 %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("解析 DeepSeek 响应失败（JSON 格式异常）: %w", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("LLM 返回内容为空，可尝试重新生成")
	}

	return chatResp.Choices[0].Message.Content, nil
}
