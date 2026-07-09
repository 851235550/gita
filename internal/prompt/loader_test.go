package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupPromptDir 在临时目录下创建 ~/.gita/prompts/ 结构并设置 HOME 环境变量。
func setupPromptDir(t *testing.T, templateName, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	promptDir := filepath.Join(tmpDir, ".gita", "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatalf("创建测试 prompts 目录失败: %v", err)
	}

	if content != "" || templateName != "" {
		templateFile := filepath.Join(promptDir, templateName+".md")
		if err := os.WriteFile(templateFile, []byte(content), 0644); err != nil {
			t.Fatalf("写入测试模板失败: %v", err)
		}
	}

	t.Setenv("HOME", tmpDir)
	return tmpDir
}

// ============================================================================
// TC-T1.3-01: ~/.gita/prompts/commit.md 不存在 → 使用内置默认模板
// ============================================================================
func TestLoadTemplate_NoUserTemplate_UsesBuiltin(t *testing.T) {
	// Arrange: 不创建用户自定义模板。
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Act
	tmpl, err := LoadTemplate("commit")

	// Assert
	if err != nil {
		t.Fatalf("加载内置模板不应报错: %v", err)
	}
	if tmpl == "" {
		t.Error("内置模板不应为空")
	}
	// 内置模板应包含关键变量占位符。
	if !strings.Contains(tmpl, "{{diff}}") {
		t.Error("内置 commit 模板应包含 {{diff}} 变量")
	}
}

// ============================================================================
// TC-T1.3-02: ~/.gita/prompts/commit.md 存在 → 优先使用用户自定义模板
// ============================================================================
func TestLoadTemplate_UserTemplateExists_UsesUserTemplate(t *testing.T) {
	// Arrange
	customContent := "CUSTOM: {{diff}}"
	setupPromptDir(t, "commit", customContent)

	// Act
	tmpl, err := LoadTemplate("commit")

	// Assert
	if err != nil {
		t.Fatalf("加载用户模板失败: %v", err)
	}
	if tmpl != customContent {
		t.Errorf("用户模板内容不匹配:\n  实际: %q\n  期望: %q", tmpl, customContent)
	}
}

// ============================================================================
// TC-T1.3-03: 模板中包含已知变量 → 全部正确替换
// ============================================================================
func TestRender_AllKnownVariables_ReplacedCorrectly(t *testing.T) {
	// Arrange
	tmpl := "Language: {{language}}, Style: {{style}}, Hint: {{hint}}, Diff: {{diff}}"
	vars := map[string]string{
		"language": "zh-CN",
		"style":    "conventional",
		"hint":     "hotfix",
		"diff":     "--- a/file.go\n+++ b/file.go",
	}

	// Act
	result := Render(tmpl, vars)

	// Assert
	if !strings.Contains(result, "Language: zh-CN") {
		t.Errorf("language 未正确替换: %s", result)
	}
	if !strings.Contains(result, "Style: conventional") {
		t.Errorf("style 未正确替换: %s", result)
	}
	if !strings.Contains(result, "Hint: hotfix") {
		t.Errorf("hint 未正确替换: %s", result)
	}
	if !strings.Contains(result, "--- a/file.go") {
		t.Errorf("diff 未正确替换: %s", result)
	}
}

// ============================================================================
// TC-T1.3-04: 未传 --hint → {{hint}} 渲染为空字符串
// ============================================================================
func TestRender_MissingHint_RendersEmpty(t *testing.T) {
	// Arrange: vars 中不包含 hint。
	tmpl := "Hint: [{{hint}}]"
	vars := map[string]string{
		"language": "zh-CN",
	}

	// Act
	result := Render(tmpl, vars)

	// Assert: hint 占位符应被替换为空，结果不含 {{hint}} 字面量。
	if strings.Contains(result, "{{hint}}") {
		t.Error("未提供的变量应渲染为空，不应保留 {{hint}} 原样")
	}
	// 期望输出: "Hint: []"
	expected := "Hint: []"
	if result != expected {
		t.Errorf("结果 = %q, want %q", result, expected)
	}
}

// ============================================================================
// TC-T1.3-05: 模板中包含未定义变量 → 渲染为空字符串
// ============================================================================
func TestRender_UnknownVariable_RendersEmpty(t *testing.T) {
	// Arrange
	tmpl := "Value: [{{unknown_var}}]"
	vars := map[string]string{
		"language": "zh-CN",
	}

	// Act
	result := Render(tmpl, vars)

	// Assert
	if strings.Contains(result, "{{unknown_var}}") {
		t.Error("未定义变量应渲染为空，不应保留原占位符")
	}
	expected := "Value: []"
	if result != expected {
		t.Errorf("结果 = %q, want %q", result, expected)
	}
}

// ============================================================================
// TC-T1.3-06: 用户自定义模板为空文件 → 视为合法，渲染为空
// ============================================================================
func TestLoadTemplate_EmptyUserTemplate_ReturnsEmpty(t *testing.T) {
	// Arrange: 创建空模板文件。注意 setupPromptDir 中空 content 仍会创建文件。
	customContent := ""
	setupPromptDir(t, "commit", customContent)

	// Act
	tmpl, err := LoadTemplate("commit")

	// Assert: 空文件不算错误，视为用户有意使用空模板。
	if err != nil {
		t.Fatalf("空模板不应报错: %v", err)
	}
	if tmpl != "" {
		t.Errorf("空模板文件应返回空字符串，实际: %q", tmpl)
	}
}

// ============================================================================
// 补充: 多变量嵌套场景
// ============================================================================
func TestRender_MultipleOccurrences_SameReplacement(t *testing.T) {
	// Arrange: 同一变量在模板中出现多次。
	tmpl := "{{diff}} and also {{diff}}"
	vars := map[string]string{
		"diff": "CONTENT",
	}

	// Act
	result := Render(tmpl, vars)

	// Assert: 两处都应被替换。
	expected := "CONTENT and also CONTENT"
	if result != expected {
		t.Errorf("结果 = %q, want %q", result, expected)
	}
}

// ============================================================================
// 补充: 不存在的子命令模板
// ============================================================================
func TestLoadTemplate_UnknownCommand_ReturnsError(t *testing.T) {
	// Arrange: 请求一个不存在的模板名。
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Act
	_, err := LoadTemplate("nonexistent")

	// Assert
	if err == nil {
		t.Error("不存在的模板应返回错误")
	}
}

// ============================================================================
// 补充: stat 和 file_list 变量
// ============================================================================
func TestRender_StatAndFileList_VariablesReplaced(t *testing.T) {
	// Arrange
	tmpl := "Stats:\n{{stat}}\n\nFiles:\n{{file_list}}"
	vars := map[string]string{
		"stat":      "file.go | 3 +++",
		"file_list": "file.go\nutil.go",
	}

	// Act
	result := Render(tmpl, vars)

	// Assert
	if !strings.Contains(result, "file.go | 3 +++") {
		t.Errorf("stat 未正确替换: %s", result)
	}
	if !strings.Contains(result, "file.go\nutil.go") {
		t.Errorf("file_list 未正确替换: %s", result)
	}
}
