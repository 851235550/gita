// Package context 负责根据 staged diff 行数进行分级判断与摘要压缩，
// 是整个 gita 中最核心的业务逻辑模块，分级边界值需精确覆盖（开发规范 1.5 节要求 100% 分支覆盖）。
package context

// DiffLevel 表示 diff 的分级结果，用于决定发送给 LLM 的内容策略。
type DiffLevel int

const (
	// Small 小型变更（< 1000 行）：发送完整 diff。
	Small DiffLevel = iota

	// Medium 中型变更（1000 ~ maxDiffLines-1 行）：发送完整 diff + --stat。
	Medium

	// Large 大型变更（maxDiffLines ~ 14999 行）：按文件摘要压缩。
	Large

	// ExtraLarge 超大型变更（>= 15000 行）：仅 --stat + --name-only，提示拆分。
	ExtraLarge
)

// 分级阈值常量。
// maxDiffLines（Medium/Large 边界）由用户通过 config.yaml 配置，默认 5000。
// 其余边界值为固定阈值，参考需求文档 6.4 节。
const (
	// smallThreshold Small 分级的上边界（不含），< 此值为 Small。
	smallThreshold = 1000

	// extraLargeThreshold ExtraLarge 分级的下边界（含），>= 此值为 ExtraLarge。
	extraLargeThreshold = 15000
)

// ClassifyDiff 根据 diff 行数和用户配置的 maxDiffLines 返回分级结果。
// maxDiffLines 对应 config.yaml 中的 max_diff_lines 字段（Medium/Large 边界，默认 5000）。
//
// 分级规则（边界值采用左闭右开约定，注释中已标注）：
//   - Small:      lineCount < 1000
//   - Medium:     1000 <= lineCount < maxDiffLines
//   - Large:      maxDiffLines <= lineCount < 15000
//   - ExtraLarge: lineCount >= 15000
//
// 若 maxDiffLines <= 0（配置异常），回退为默认值 5000，确保不会出现负数边界。
func ClassifyDiff(lineCount int, maxDiffLines int) DiffLevel {
	if maxDiffLines <= 0 {
		// 防御性处理：配置异常时回退默认值，避免分级逻辑出错。
		maxDiffLines = 5000
	}

	// 注意：ExtraLarge（>= 15000）需在 Medium 判断之前检查，
	// 因为当用户将 maxDiffLines 设置得很大（如 20000）时，
	// 18000 行可能同时满足 < maxDiffLines 和 >= extraLargeThreshold，
	// ExtraLarge 应具有更高优先级 —— 超大型 diff 无论如何都应触发拆分提示。
	switch {
	case lineCount < smallThreshold:
		return Small
	case lineCount >= extraLargeThreshold:
		return ExtraLarge
	case lineCount < maxDiffLines:
		return Medium
	default:
		return Large
	}
}
