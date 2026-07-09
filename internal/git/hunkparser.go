package git

import (
	"regexp"
	"strings"
)

// hunkHeaderPattern 匹配 git diff 中的 hunk header 行，提取末尾的函数/类上下文文本。
// Git 对主流语言（C/C++、Python、Java、Go、Rust、JS/TS 等）内置了 funcname 识别规则，
// 会在 @@ 行尾自动附加当前函数/类名作为上下文提示。
// 示例输入: "@@ -10,7 +10,8 @@ func doSomething(ctx context.Context) error {"
// 提取结果: "func doSomething(ctx context.Context) error {"
// 参考需求文档 6.5 节。
var hunkHeaderPattern = regexp.MustCompile(`^@@ .*@@ ?(.*)$`)

// diffFilePattern 匹配 diff 中的文件路径标记行，
// 用于跟踪当前正在处理的文件。
// 示例: "diff --git a/internal/git/diff.go b/internal/git/diff.go"
var diffFilePattern = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)

// ExtractHunkContexts 从 git diff 文本中提取每个文件的 hunk 上下文信息。
// 返回值 map 的 key 为文件路径，value 为该文件中所有 hunk 的上下文文本列表（已去重）。
//
// Git 对部分小众语言（如 .proto）没有内置 funcname 识别规则，
// 此时 hunk header 末尾无上下文文本，该 hunk 对应的 value 为空字符串。
// 上层（context 模块）会自动降级为"仅文件名+行数统计"展示，详见需求文档 6.5 节。
func ExtractHunkContexts(diffText string) map[string][]string {
	if diffText == "" {
		return nil
	}

	result := make(map[string][]string)
	lines := strings.Split(diffText, "\n")

	var currentFile string

	for _, line := range lines {
		// 检测文件切换。
		if matches := diffFilePattern.FindStringSubmatch(line); matches != nil {
			// matches[1] 是 a/ 前缀的路径，matches[2] 是 b/ 前缀的路径，
			// 两者通常相同（非重命名场景），取 b/ 路径作为当前文件标识。
			currentFile = matches[2]
			continue
		}

		// 检测 hunk header 行。
		if matches := hunkHeaderPattern.FindStringSubmatch(line); matches != nil {
			context := strings.TrimSpace(matches[1])
			if currentFile == "" {
				// 理论上不应出现（diff 中先有文件路径才有 hunk），防御性跳过。
				continue
			}

			// 去重：同一文件内多个 hunk 落在同一函数时只保留一条。
			if !contains(result[currentFile], context) {
				result[currentFile] = append(result[currentFile], context)
			}
		}
	}

	return result
}

// contains 判断字符串切片中是否已包含指定字符串。
// 注意：空字符串不会被去重（需求：空上下文是合法的降级结果，允许重复出现）。
func contains(slice []string, s string) bool {
	if s == "" {
		// 空上下文不参与去重 —— 它本身就是"无信息"的标记，
		// 去重无意义且会掩盖文件中存在多个无上下文 hunk 的事实。
		return false
	}
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
