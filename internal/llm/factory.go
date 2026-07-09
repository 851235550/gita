package llm

import (
	"fmt"

	"gita/internal/config"
)

// NewProvider 根据配置和 provider 名称创建对应的 Provider 实例。
// providerName 为 config.yaml 中 providers 下的 key（如 "deepseek"、"openai"）。
// apiKeyOverride 为命令行 --api-key 传入的临时 Key，优先级最高。
//
// 若指定 provider 未在配置中找到，返回错误并提示可用 provider 列表。
func NewProvider(cfg *config.Config, providerName string, apiKeyOverride string) (Provider, error) {
	providerCfg, ok := cfg.Providers[providerName]
	if !ok {
		// 列出所有已配置 provider 名称，帮助用户排查。
		available := make([]string, 0, len(cfg.Providers))
		for name := range cfg.Providers {
			available = append(available, name)
		}
		return nil, fmt.Errorf("未找到 provider '%s' 的配置，可用 provider: %v", providerName, available)
	}

	switch providerName {
	case "deepseek":
		return NewDeepSeekProvider(&providerCfg, apiKeyOverride)
	case "openai":
		return NewOpenAIProvider(&providerCfg, apiKeyOverride)
	case "claude":
		return NewClaudeProvider(&providerCfg, apiKeyOverride)
	default:
		// 未知 provider 名称（未在 factory 中注册）。
		return nil, fmt.Errorf("不支持的 provider: %s（当前支持: deepseek, openai, claude）", providerName)
	}
}
