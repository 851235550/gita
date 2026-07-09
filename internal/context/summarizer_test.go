package context

import (
	"strings"
	"testing"
)

// ============================================================================
// TC-T2.2-01: 混合场景 —— 小文件全量 diff，大文件摘要
// ============================================================================
func TestBuildSummary_TC_T2_2_01_MixedFiles(t *testing.T) {
	// Arrange: 文件 A 改动 < 50 行，文件 B 改动 >= 50 行。
	files := []FileChange{
		{
			Path:         "small.go",
			ChangeType:   ChangeModified,
			LinesAdded:   3,
			LinesDeleted: 1, // 总计 4 行 < 50。
			FullDiff:     "--- small.go\n+++ small.go\n@@ -1,3 +1,3 @@\n-old\n+new",
		},
		{
			Path:         "large.go",
			ChangeType:   ChangeModified,
			LinesAdded:   60,
			LinesDeleted: 10, // 总计 70 行 >= 50。
			HunkContexts: []string{"func doWork() {"},
			FullDiff:     "FULL DIFF OF LARGE FILE",
		},
	}

	// Act
	result := BuildSummary(files)

	// Assert: 小文件应保留完整 diff。
	if !strings.Contains(result, "--- small.go") {
		t.Error("小文件的完整 diff 应保留在输出中")
	}
	if !strings.Contains(result, "-old") {
		t.Error("小文件的变更内容应保留")
	}

	// 大文件应为摘要形式，不应包含完整 diff。
	if strings.Contains(result, "FULL DIFF OF LARGE FILE") {
		t.Error("大文件不应包含完整 diff，应走摘要压缩")
	}
	if !strings.Contains(result, "large.go (修改)") {
		t.Error("大文件摘要应包含文件路径和变更类型")
	}
	if !strings.Contains(result, "+60") {
		t.Error("大文件摘要应包含新增行数")
	}
	if !strings.Contains(result, "-10") {
		t.Error("大文件摘要应包含删除行数")
	}
	if !strings.Contains(result, "func doWork()") {
		t.Error("大文件摘要应包含 hunk 上下文")
	}
}

// ============================================================================
// TC-T2.2-02: 文件改动恰好 50 行 → 走摘要压缩（50 不满足 "< 50"）
// ============================================================================
func TestBuildSummary_TC_T2_2_02_Exactly50Lines_Summary(t *testing.T) {
	// Arrange: 总变更恰好 50 行，边界定义是 < 50 才走全量，>= 50 走摘要。
	files := []FileChange{
		{
			Path:         "exact.go",
			ChangeType:   ChangeModified,
			LinesAdded:   30,
			LinesDeleted: 20, // 总计 50 行，不满足 < 50 条件。
			FullDiff:     "FULL DIFF",
		},
	}

	// Act
	result := BuildSummary(files)

	// Assert: 50 行应走摘要压缩。
	if strings.Contains(result, "FULL DIFF") {
		t.Error("恰好 50 行应走摘要压缩，不应包含完整 diff")
	}
	if !strings.Contains(result, "exact.go") {
		t.Error("摘要应包含文件路径")
	}
}

// ============================================================================
// TC-T2.2-03: 新增文件 → 变更类型标注为"新增"
// ============================================================================
func TestBuildSummary_TC_T2_2_03_NewFile_Added(t *testing.T) {
	// Arrange
	files := []FileChange{
		{
			Path:         "newfile.go",
			ChangeType:   ChangeAdded,
			LinesAdded:   100,
			LinesDeleted: 0,
			HunkContexts: []string{"func NewFunc() {"},
		},
	}

	// Act
	result := BuildSummary(files)

	// Assert
	if !strings.Contains(result, "(新增)") {
		t.Error("新增文件的摘要应标注 '(新增)'")
	}
	if !strings.Contains(result, "func NewFunc()") {
		t.Error("新增文件的摘要应包含 hunk 上下文")
	}
}

// ============================================================================
// TC-T2.2-04: 删除文件 → 变更类型标注为"删除"
// ============================================================================
func TestBuildSummary_TC_T2_2_04_DeletedFile_Deleted(t *testing.T) {
	// Arrange
	files := []FileChange{
		{
			Path:         "oldfile.go",
			ChangeType:   ChangeDeleted,
			LinesAdded:   0,
			LinesDeleted: 80,
		},
	}

	// Act
	result := BuildSummary(files)

	// Assert: 删除文件无 hunk 上下文（无新增内容可解析）。
	if !strings.Contains(result, "(删除)") {
		t.Error("删除文件的摘要应标注 '(删除)'")
	}
	if !strings.Contains(result, "+0 行, -80 行") {
		t.Error("删除文件应包含行数统计")
	}
	if strings.Contains(result, "改动范围") {
		t.Error("删除文件不应有 hunk 上下文（无新增内容）")
	}
}

// ============================================================================
// TC-T2.2-05: 重命名文件 → 摘要场景下标注旧路径 → 新路径
// ============================================================================
func TestBuildSummary_TC_T2_2_05_RenamedFile(t *testing.T) {
	// Arrange: 重命名 + 大量内容变更（>= 50 行），触发摘要路径。
	files := []FileChange{
		{
			Path:         "new_name.go",
			OldPath:      "old_name.go",
			ChangeType:   ChangeRenamed,
			LinesAdded:   60,
			LinesDeleted: 0,
		},
	}

	// Act
	result := BuildSummary(files)

	// Assert: 摘要应包含重命名标记。
	if !strings.Contains(result, "(重命名: old_name.go → new_name.go)") {
		t.Errorf("重命名摘要应标注路径变更，实际输出: %s", result)
	}
}

// ============================================================================
// 补充: 空文件列表
// ============================================================================
func TestBuildSummary_EmptyFiles_ReturnsEmpty(t *testing.T) {
	// Arrange

	// Act
	result := BuildSummary(nil)

	// Assert
	if result != "" {
		t.Errorf("空文件列表应返回空字符串，实际: %q", result)
	}
}

// ============================================================================
// 补充: ParseDiffFiles 解析完整 diff
// ============================================================================
func TestParseDiffFiles_ModifiedFile(t *testing.T) {
	// Arrange: 模拟 git diff --staged 输出。
	diff := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
-old line
+new line
 unchanged`

	// Act
	files := ParseDiffFiles(diff)

	// Assert
	if len(files) != 1 {
		t.Fatalf("期望 1 个文件，实际: %d", len(files))
	}
	f := files[0]
	if f.Path != "main.go" {
		t.Errorf("Path = %q, want main.go", f.Path)
	}
	if f.ChangeType != ChangeModified {
		t.Errorf("ChangeType = %v, want ChangeModified", f.ChangeType)
	}
}

// ============================================================================
// 补充: ParseDiffFiles 新增文件
// ============================================================================
func TestParseDiffFiles_NewFile(t *testing.T) {
	// Arrange
	diff := `diff --git a/new.go b/new.go
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+line1
+line2
+line3`

	// Act
	files := ParseDiffFiles(diff)

	// Assert
	if len(files) != 1 {
		t.Fatalf("期望 1 个文件，实际: %d", len(files))
	}
	if files[0].ChangeType != ChangeAdded {
		t.Errorf("新增文件 ChangeType 应为 ChangeAdded，实际: %v", files[0].ChangeType)
	}
}

// ============================================================================
// 补充: ParseDiffFiles 删除文件
// ============================================================================
func TestParseDiffFiles_DeletedFile(t *testing.T) {
	// Arrange
	diff := `diff --git a/old.go b/old.go
deleted file mode 100644
index abc1234..0000000
--- a/old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-line1
-line2
-line3`

	// Act
	files := ParseDiffFiles(diff)

	// Assert
	if len(files) != 1 {
		t.Fatalf("期望 1 个文件，实际: %d", len(files))
	}
	if files[0].ChangeType != ChangeDeleted {
		t.Errorf("删除文件 ChangeType 应为 ChangeDeleted，实际: %v", files[0].ChangeType)
	}
}

// ============================================================================
// 补充: ParseDiffFiles 空输入
// ============================================================================
func TestParseDiffFiles_EmptyInput(t *testing.T) {
	// Arrange

	// Act
	files := ParseDiffFiles("")

	// Assert
	if files != nil {
		t.Errorf("空输入应返回 nil，实际: %v", files)
	}
}

// ============================================================================
// 补充: parseHunkLineCounts 正常解析
// ============================================================================
func TestParseHunkLineCounts_Normal(t *testing.T) {
	// Arrange
	line := "@@ -10,7 +10,8 @@ func main() {"

	// Act
	added, deleted := parseHunkLineCounts(line)

	// Assert
	if added != 8 {
		t.Errorf("added = %d, want 8", added)
	}
	if deleted != 7 {
		t.Errorf("deleted = %d, want 7", deleted)
	}
}

// ============================================================================
// 补充: parseHunkLineCounts 异常输入
// ============================================================================
func TestParseHunkLineCounts_Invalid(t *testing.T) {
	// Arrange
	line := "not a hunk header"

	// Act
	added, deleted := parseHunkLineCounts(line)

	// Assert: 解析失败返回 (0, 0)，不 panic。
	if added != 0 || deleted != 0 {
		t.Errorf("异常输入应返回 (0, 0)，实际: (%d, %d)", added, deleted)
	}
}
