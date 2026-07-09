package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupConfigDir 在临时目录下创建 ~/.gita/config.yaml 结构并设置 HOME 环境变量。
// 返回临时目录路径和清理函数。
func setupConfigDir(t *testing.T, yamlContent string) string {
	t.Helper()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".gita")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建测试配置目录失败: %v", err)
	}

	if yamlContent != "" {
		configFile := filepath.Join(configDir, "config.yaml")
		if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("写入测试配置文件失败: %v", err)
		}
	}

	t.Setenv("HOME", tmpDir)
	return tmpDir
}

// ============================================================================
// TC-T0.2-01: 配置文件不存在时使用内置默认值，不报错
// ============================================================================
func TestLoad_NoConfigFile_UsesDefaults(t *testing.T) {
	// Arrange: 不创建任何配置文件，仅设置 HOME。
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Act
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("配置文件不存在时不应报错，但返回了 error: %v", err)
	}
	if cfg == nil {
		t.Fatal("返回的 Config 不应为 nil")
	}
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider 默认值应为 deepseek，实际: %s", cfg.DefaultProvider)
	}
	if cfg.Language != "zh-CN" {
		t.Errorf("Language 默认值应为 zh-CN，实际: %s", cfg.Language)
	}
	if cfg.CommitStyle != "conventional" {
		t.Errorf("CommitStyle 默认值应为 conventional，实际: %s", cfg.CommitStyle)
	}
	if cfg.MaxDiffLines != 5000 {
		t.Errorf("MaxDiffLines 默认值应为 5000，实际: %d", cfg.MaxDiffLines)
	}
	if len(cfg.Providers) == 0 {
		t.Error("Providers 默认值不应为空")
	}
}

// ============================================================================
// TC-T0.2-02: 完整配置文件，所有字段均正确解析
// ============================================================================
func TestLoad_FullConfig_ParsesCorrectly(t *testing.T) {
	// Arrange: 构造一份包含全部字段的合法 YAML。
	setupConfigDir(t, `
default_provider: openai
language: en
commit_style: plain
max_diff_lines: 3000
providers:
  openai:
    model: gpt-4o-mini
    base_url: https://api.openai.com/v1
    api_key_env: GITA_OPENAI_API_KEY
  deepseek:
    model: deepseek-chat
    base_url: https://api.deepseek.com/v1
    api_key_env: GITA_DEEPSEEK_API_KEY
`)

	// Act
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("合法配置文件解析失败: %v", err)
	}
	if cfg.DefaultProvider != "openai" {
		t.Errorf("DefaultProvider = %s, want openai", cfg.DefaultProvider)
	}
	if cfg.Language != "en" {
		t.Errorf("Language = %s, want en", cfg.Language)
	}
	if cfg.CommitStyle != "plain" {
		t.Errorf("CommitStyle = %s, want plain", cfg.CommitStyle)
	}
	if cfg.MaxDiffLines != 3000 {
		t.Errorf("MaxDiffLines = %d, want 3000", cfg.MaxDiffLines)
	}
	if len(cfg.Providers) != 2 {
		t.Fatalf("Providers 数量 = %d, want 2", len(cfg.Providers))
	}
	if cfg.Providers["openai"].Model != "gpt-4o-mini" {
		t.Errorf("openai model = %s, want gpt-4o-mini", cfg.Providers["openai"].Model)
	}
	if cfg.Providers["deepseek"].APIKeyEnv != "GITA_DEEPSEEK_API_KEY" {
		t.Errorf("deepseek api_key_env = %s, want GITA_DEEPSEEK_API_KEY", cfg.Providers["deepseek"].APIKeyEnv)
	}
}

// ============================================================================
// TC-T0.2-03: 部分字段缺失 —— 缺失字段使用默认值
// ============================================================================
func TestLoad_PartialConfig_FillsDefaults(t *testing.T) {
	// Arrange: 只配置 language，其余字段留空。
	setupConfigDir(t, `
language: en
`)

	// Act
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("部分字段缺失的配置解析失败: %v", err)
	}
	// 明确配置的字段应生效。
	if cfg.Language != "en" {
		t.Errorf("Language = %s, want en", cfg.Language)
	}
	// 未配置的字段应使用默认值。
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider 应使用默认值 deepseek，实际: %s", cfg.DefaultProvider)
	}
	if cfg.CommitStyle != "conventional" {
		t.Errorf("CommitStyle 应使用默认值 conventional，实际: %s", cfg.CommitStyle)
	}
	if cfg.MaxDiffLines != 5000 {
		t.Errorf("MaxDiffLines 应使用默认值 5000，实际: %d", cfg.MaxDiffLines)
	}
	// Providers 未配置时也应使用内置默认值。
	if len(cfg.Providers) == 0 {
		t.Error("Providers 未配置时应使用内置默认值")
	}
}

// ============================================================================
// TC-T0.2-04: 空配置文件（文件存在但内容为空）
// ============================================================================
func TestLoad_EmptyConfigFile(t *testing.T) {
	// Arrange: 创建空配置文件。
	setupConfigDir(t, "")

	// Act
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("空配置文件解析不应报错: %v", err)
	}
	// 空文件解析后所有字段均为零值，mergeConfig 不会覆盖，应全部为默认值。
	if cfg.DefaultProvider != "deepseek" {
		t.Errorf("DefaultProvider = %s, want deepseek", cfg.DefaultProvider)
	}
	if cfg.Language != "zh-CN" {
		t.Errorf("Language = %s, want zh-CN", cfg.Language)
	}
	if cfg.CommitStyle != "conventional" {
		t.Errorf("CommitStyle = %s, want conventional", cfg.CommitStyle)
	}
	if cfg.MaxDiffLines != 5000 {
		t.Errorf("MaxDiffLines = %d, want 5000", cfg.MaxDiffLines)
	}
}

// ============================================================================
// TC-T0.2-05: 非法 YAML 格式 —— 应返回错误
// ============================================================================
func TestLoad_InvalidYAML_ReturnsError(t *testing.T) {
	// Arrange: 写入非法的 YAML 内容（缩进混乱）。
	setupConfigDir(t, `default_provider: [unclosed bracket`)

	// Act
	_, err := Load()

	// Assert
	if err == nil {
		t.Error("非法 YAML 应返回错误，但返回了 nil")
	}
}

// ============================================================================
// TC-T0.2-06: Provider 配置覆盖 —— 用户自定义 provider 完全替换默认值
// ============================================================================
func TestLoad_CustomProviders_ReplacesDefaults(t *testing.T) {
	// Arrange: 只配置自定义 provider，不包含内置的三个。
	setupConfigDir(t, `
providers:
  custom_llm:
    model: my-model
    base_url: https://custom.api.com/v1
    api_key_env: CUSTOM_KEY
`)

	// Act
	cfg, err := Load()

	// Assert
	if err != nil {
		t.Fatalf("自定义 provider 配置解析失败: %v", err)
	}
	// 用户配置的 provider 表完全覆盖默认表（需求：让用户有完全控制权）。
	if len(cfg.Providers) != 1 {
		t.Fatalf("Providers 数量 = %d, want 1（用户配置应完全覆盖默认值）", len(cfg.Providers))
	}
	if _, ok := cfg.Providers["custom_llm"]; !ok {
		t.Error("应包含用户自定义的 custom_llm provider")
	}
	if _, ok := cfg.Providers["deepseek"]; ok {
		t.Error("用户未配置 deepseek 时不应出现在结果中")
	}
}
