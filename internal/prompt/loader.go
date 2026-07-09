// Package prompt 负责 Commit Message 生成模板的加载与变量渲染。
// 模板加载优先级：~/.gita/prompts/<name>.md（用户自定义）> 内置默认模板。
// 模板语法使用 {{variable}} 占位符，支持的变量见需求文档 7.4 节。
package prompt

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed templates/commit.md
var defaultCommitTemplate string

// 内置模板表：key 为模板名称（对应子命令），value 为模板内容。
var builtinTemplates = map[string]string{
	"commit": defaultCommitTemplate,
}

// LoadTemplate 加载指定名称的模板文件。
// 优先从 ~/.gita/prompts/<name>.md 加载，不存在时回退到内置模板。
// name 参数为子命令名（如 "commit"、"pr"），不含扩展名。
func LoadTemplate(name string) (string, error) {
	userPath, err := userTemplatePath(name)
	if err != nil {
		return "", fmt.Errorf("无法定位用户模板目录: %w", err)
	}

	data, err := os.ReadFile(userPath)
	if err == nil {
		// 用户自定义模板存在时直接使用，即使为空文件也视为有意为之。
		// 空模板意味着用户想发送空 prompt，不自动回退到内置模板。
		return string(data), nil
	}

	if !os.IsNotExist(err) {
		// 文件存在但读取失败（如权限问题），应报错而非静默回退。
		return "", fmt.Errorf("读取用户模板 %s 失败: %w", userPath, err)
	}

	// 用户模板不存在，使用内置默认模板。
	if tmpl, ok := builtinTemplates[name]; ok {
		return tmpl, nil
	}

	return "", fmt.Errorf("未找到模板: %s（既无用户自定义模板，也无内置模板）", name)
}

// Render 使用 vars 中的值替换模板中的 {{variable}} 占位符。
// 支持的变量：diff、stat、file_list、hint、language、style（见需求文档 7.4 节）。
//
// 未在 vars 中提供的变量渲染为空字符串，不会保留 {{var}} 原样。
// 模板中出现的未定义变量也渲染为空字符串，不报错。
// 这样的设计保证即使模板引用了用户未传入的变量，输出也不会出现原始占位符。
func Render(template string, vars map[string]string) string {
	// 使用 regex 而非 strings.Replacer，因为需要处理 {{unknown}} 这种不在替换列表中的变量。
	result := template
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 清除所有未被替换的 {{...}} 占位符（包括未知变量和拼写错误）。
	// 正则匹配 {{任意非 } 字符}}，确保不会贪婪匹配跨多个占位符。
	unknownPattern := regexp.MustCompile(`\{\{[^}]+\}\}`)
	result = unknownPattern.ReplaceAllString(result, "")

	return result
}

// userTemplatePath 返回 ~/.gita/prompts/<name>.md 的绝对路径。
func userTemplatePath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gita", "prompts", name+".md"), nil
}
