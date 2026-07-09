package context

import (
	"fmt"
	"strings"

	"gita/internal/git"
)

// ChangeType 表示文件变更类型。
type ChangeType int

const (
	ChangeModified ChangeType = iota // 修改
	ChangeAdded                      // 新增
	ChangeDeleted                    // 删除
	ChangeRenamed                    // 重命名
)

// FileChange 描述单个文件在本次 staged diff 中的变更信息。
type FileChange struct {
	// Path 文件路径（使用 b/ 路径，即新路径）。
	Path string

	// OldPath 重命名前的路径（仅重命名场景非空）。
	OldPath string

	// ChangeType 变更类型（新增/修改/删除/重命名）。
	ChangeType ChangeType

	// LinesAdded 新增行数。
	LinesAdded int

	// LinesDeleted 删除行数。
	LinesDeleted int

	// HunkContexts 该文件所有 hunk 的上下文文本（已去重）。
	HunkContexts []string

	// FullDiff 该文件在 diff 中的完整片段。
	FullDiff string
}

// 摘要压缩阈值：单文件改动 < 此值时不压缩，保留完整 diff。
// 参考需求文档 6.5 节"抓大放小"原则。
const smallFileThreshold = 50

// BuildSummary 对变更文件列表执行"抓大放小"策略：
// 单文件改动 < 50 行 → 保留完整 diff；
// 单文件改动 >= 50 行 → 输出"路径 + 变更类型 + 行数统计 + hunk 上下文列表"的摘要形式。
//
// 返回值可直接拼入 prompt 的 {{diff}} 变量。
func BuildSummary(files []FileChange) string {
	var sb strings.Builder

	for i, f := range files {
		if i > 0 {
			sb.WriteString("\n")
		}

		totalChanges := f.LinesAdded + f.LinesDeleted
		if totalChanges < smallFileThreshold {
			// 小文件不压缩，直接发送完整 diff。
			sb.WriteString(f.FullDiff)
		} else {
			// 大文件走摘要压缩。
			sb.WriteString(buildFileSummary(f))
		}
	}

	return sb.String()
}

// buildFileSummary 构造单个文件的摘要描述。
func buildFileSummary(f FileChange) string {
	var sb strings.Builder

	// 文件路径与变更类型。
	sb.WriteString(fmt.Sprintf("--- 文件: %s", f.Path))
	switch f.ChangeType {
	case ChangeAdded:
		sb.WriteString(" (新增)")
	case ChangeDeleted:
		sb.WriteString(" (删除)")
	case ChangeRenamed:
		sb.WriteString(fmt.Sprintf(" (重命名: %s → %s)", f.OldPath, f.Path))
	case ChangeModified:
		sb.WriteString(" (修改)")
	}
	sb.WriteString(" ---\n")

	// 增删行数统计。
	sb.WriteString(fmt.Sprintf("  +%d 行, -%d 行\n", f.LinesAdded, f.LinesDeleted))

	// Hunk 上下文列表（改动涉及的函数/类）。
	if len(f.HunkContexts) > 0 {
		sb.WriteString("  改动范围:\n")
		for _, ctx := range f.HunkContexts {
			if ctx != "" {
				sb.WriteString(fmt.Sprintf("    - %s\n", ctx))
			}
		}
	}

	return sb.String()
}

// ParseDiffFiles 将完整的 git diff --staged 输出解析为 FileChange 列表。
// 用于后续的"抓大放小"摘要压缩。
//
// 解析逻辑基于 git diff 的标准 unified diff 格式，
// 识别文件头（diff --git / --- / +++）、变更类型、hunk header 和行数统计。
func ParseDiffFiles(diffText string) []FileChange {
	if diffText == "" {
		return nil
	}

	lines := strings.Split(diffText, "\n")
	var files []FileChange
	var current *FileChange
	var inFile bool

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			// 新文件开始，保存上一个文件。
			if current != nil {
				files = append(files, *current)
			}
			current = &FileChange{
				ChangeType: ChangeModified, // 默认修改，后续根据具体标记调整。
			}
			inFile = true

			// 解析 diff --git a/old b/new 中的路径。
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				current.OldPath = strings.TrimPrefix(parts[2], "a/")
				current.Path = strings.TrimPrefix(parts[3], "b/")
			}

		case strings.HasPrefix(line, "new file mode"):
			if current != nil {
				current.ChangeType = ChangeAdded
			}

		case strings.HasPrefix(line, "deleted file mode"):
			if current != nil {
				current.ChangeType = ChangeDeleted
			}

		case strings.HasPrefix(line, "rename from "):
			if current != nil {
				current.ChangeType = ChangeRenamed
			}

		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			// 文件路径行，不统计行数。
			continue

		case strings.HasPrefix(line, "@@") && inFile && current != nil:
			// Hunk header：解析行数统计和上下文。
			added, deleted := parseHunkLineCounts(line)
			current.LinesAdded += added
			current.LinesDeleted += deleted

		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") && inFile:
			if current != nil {
				current.LinesAdded++
			}

		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") && inFile:
			if current != nil {
				current.LinesDeleted++
			}
		}

		// 累积当前文件的完整 diff 片段。
		if current != nil {
			if current.FullDiff != "" {
				current.FullDiff += "\n"
			}
			current.FullDiff += line
		}
	}

	// 保存最后一个文件。
	if current != nil {
		files = append(files, *current)
	}

	// 补充 hunk 上下文信息。
	hunkContexts := git.ExtractHunkContexts(diffText)
	for i := range files {
		if ctxs, ok := hunkContexts[files[i].Path]; ok {
			files[i].HunkContexts = ctxs
		}
	}

	return files
}

// parseHunkLineCounts 从 hunk header 行解析增删行数。
// 示例: "@@ -10,7 +10,8 @@" → added=8, deleted=7
// 若解析失败则返回 (0, 0)，不阻塞主流程。
func parseHunkLineCounts(line string) (added, deleted int) {
	// 取 @@ 之间的内容。
	parts := strings.SplitN(line, "@@", 3)
	if len(parts) < 3 {
		return 0, 0
	}
	content := strings.TrimSpace(parts[1])

	// 按空格分割，形如 "-10,7 +10,8"。
	fields := strings.Fields(content)
	for _, field := range fields {
		if strings.HasPrefix(field, "+") {
			nums := strings.Split(strings.TrimPrefix(field, "+"), ",")
			if len(nums) >= 2 {
				fmt.Sscanf(nums[1], "%d", &added)
			}
		}
		if strings.HasPrefix(field, "-") {
			nums := strings.Split(strings.TrimPrefix(field, "-"), ",")
			if len(nums) >= 2 {
				fmt.Sscanf(nums[1], "%d", &deleted)
			}
		}
	}

	return added, deleted
}
