package context

import (
	"fmt"
	"strings"
)

// BuildContext 根据 diff 分级结果，拼装最终发送给 LLM 的上下文文本。
// level 由 ClassifyDiff 预先计算得出，本函数不重复判断分级逻辑。
// diff 为完整 git diff --staged 输出，stat 为 --stat 摘要，fileNames 为变更文件列表。
//
// 当 level 为 ExtraLarge 且 needConfirm 为 true 时，
// 调用方需自行处理用户交互确认，本函数只负责内容拼装，不包含任何 I/O 操作。
//
// 返回值内容对应需求文档 6.4 节表格中各分级档位的发送内容定义：
//   - Small:     仅完整 diff
//   - Medium:    完整 diff + --stat
//   - Large:     摘要压缩结果（含"抓大放小"）
//   - ExtraLarge: --stat + --name-only，附拆分建议
func BuildContext(diff, stat string, fileNames []string, level DiffLevel) string {
	var sb strings.Builder

	switch level {
	case Small:
		// 小型变更：直接发送完整 diff，不附加统计信息以减少 token 消耗。
		sb.WriteString(diff)

	case Medium:
		// 中型变更：完整 diff + 统计概览，帮助 LLM 快速把握整体变更结构。
		sb.WriteString("## 变更统计\n")
		sb.WriteString(stat)
		sb.WriteString("\n\n## 变更详情\n")
		sb.WriteString(diff)

	case Large:
		// 大型变更：执行摘要压缩（"抓大放小"原则，需求文档 6.5 节）。
		files := ParseDiffFiles(diff)
		summary := BuildSummary(files)
		sb.WriteString(summary)

	case ExtraLarge:
		// 超大型变更：仅发送 stat + 文件列表，不做详细 diff 分析。
		sb.WriteString("## 变更统计\n")
		sb.WriteString(stat)
		sb.WriteString("\n\n## 变更文件列表\n")
		sb.WriteString(strings.Join(fileNames, "\n"))
		sb.WriteString("\n\n")
		sb.WriteString(fmt.Sprintf(
			"注意：本次暂存变更规模较大（%d 个文件），建议拆分提交。\n"+
				"以下 Commit Message 基于文件列表粗略生成，质量可能下降。",
			len(fileNames),
		))
	}

	return sb.String()
}
