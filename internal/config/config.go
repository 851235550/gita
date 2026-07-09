// Package config 负责加载和管理 ~/.gita/config.yaml 配置文件。
// 配置文件不存在时使用内置默认值，不报错。
// 出于安全考虑，不支持在 config.yaml 中明文写 api_key，仅通过 api_key_env 指定环境变量名。
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 对应 ~/.gita/config.yaml 的完整结构，详见需求文档 7.2 节。
type Config struct {
	// DefaultProvider 默认使用的 LLM Provider 名称，对应 providers 下的 key。
	DefaultProvider string `yaml:"default_provider"`

	// Providers 可用的 LLM Provider 配置表，key 为 provider 名称（如 deepseek/openai/claude）。
	Providers map[string]ProviderConfig `yaml:"providers"`

	// Language 默认输出语言（zh-CN 或 en）。
	Language string `yaml:"language"`

	// CommitStyle Commit Message 风格（conventional 或 plain）。
	CommitStyle string `yaml:"commit_style"`

	// MaxDiffLines Diff 分级的中型阈值上限，默认 5000。
	// 分级逻辑参考需求文档 6.4 节：Small(<1000) / Medium(1000~此值) / Large(此值~15000) / ExtraLarge(>15000)。
	MaxDiffLines int `yaml:"max_diff_lines"`
}

// ProviderConfig 描述单个 LLM Provider 的连接参数，详见需求文档 7.2 节。
type ProviderConfig struct {
	// Model 模型名称（如 deepseek-chat、gpt-4o）。
	Model string `yaml:"model"`

	// BaseURL API 端点地址（如 https://api.deepseek.com/v1）。
	BaseURL string `yaml:"base_url"`

	// APIKeyEnv 存放 API Key 的环境变量名（如 GITA_DEEPSEEK_API_KEY）。
	APIKeyEnv string `yaml:"api_key_env"`
}

// 默认值常量 —— 当配置文件不存在或字段缺失时使用。
const (
	defaultProvider = "deepseek"
	defaultLanguage = "zh-CN"
	defaultStyle    = "conventional"
	defaultMaxDiff  = 5000
)

// Load 从 ~/.gita/config.yaml 加载配置。
// 配置文件不存在时返回带有默认值的 Config，不报错。
// 配置文件存在但部分字段缺失时，缺失字段使用默认值填充。
func Load() (*Config, error) {
	cfg := &Config{
		DefaultProvider: defaultProvider,
		Language:        defaultLanguage,
		CommitStyle:     defaultStyle,
		MaxDiffLines:    defaultMaxDiff,
		Providers:       defaultProviders(),
	}

	path, err := configPath()
	if err != nil {
		// 无法获取 home 目录属于极端情况，返回错误。
		return nil, fmt.Errorf("无法定位配置文件路径: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在时使用内置默认值，属于正常场景，不报错。
			return cfg, nil
		}
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	// 解析 YAML，使用独立变量避免覆盖已设置的默认值。
	var parsed Config
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败（YAML 格式可能有误）: %w", path, err)
	}

	// 按字段合并：用户配置覆盖默认值，未配置的字段保留默认值。
	mergeConfig(cfg, &parsed)

	return cfg, nil
}

// configPath 返回 ~/.gita/config.yaml 的绝对路径。
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gita", "config.yaml"), nil
}

// defaultProviders 返回内置的 Provider 默认配置，
// 确保用户在完全没有配置文件时也能知道需要设置哪些环境变量。
func defaultProviders() map[string]ProviderConfig {
	return map[string]ProviderConfig{
		"deepseek": {
			Model:     "deepseek-chat",
			BaseURL:   "https://api.deepseek.com/v1",
			APIKeyEnv: "GITA_DEEPSEEK_API_KEY",
		},
		"openai": {
			Model:     "gpt-4o",
			BaseURL:   "https://api.openai.com/v1",
			APIKeyEnv: "GITA_OPENAI_API_KEY",
		},
		"claude": {
			Model:     "claude-sonnet-4-6",
			BaseURL:   "https://api.anthropic.com/v1",
			APIKeyEnv: "GITA_ANTHROPIC_API_KEY",
		},
	}
}

// mergeConfig 将 parsed 中非零值字段覆盖到 cfg 上。
// 仅当 parsed 字段不为空/零值时覆盖，保留 cfg 已有默认值。
func mergeConfig(cfg *Config, parsed *Config) {
	if parsed.DefaultProvider != "" {
		cfg.DefaultProvider = parsed.DefaultProvider
	}
	if parsed.Language != "" {
		cfg.Language = parsed.Language
	}
	if parsed.CommitStyle != "" {
		cfg.CommitStyle = parsed.CommitStyle
	}
	if parsed.MaxDiffLines != 0 {
		cfg.MaxDiffLines = parsed.MaxDiffLines
	}
	if parsed.Providers != nil && len(parsed.Providers) > 0 {
		// 用户配置的 provider 表完全覆盖默认表，
		// 让用户有完全控制权（可增删 provider）。
		cfg.Providers = parsed.Providers
	}
}
