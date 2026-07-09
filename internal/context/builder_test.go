package context

import (
	"strings"
	"testing"
)

// ============================================================================
// TC-T2.3-01: Small 分级 → 仅完整 diff
// ============================================================================
func TestBuildContext_TC_T2_3_01_Small_OnlyDiff(t *testing.T) {
	// Arrange
	diff := "--- a/file.go\n+++ b/file.go\n@@ -1 +1 @@\n-old\n+new"
	stat := "file.go | 1 +"
	fileNames := []string{"file.go"}

	// Act
	result := BuildContext(diff, stat, fileNames, Small)

	// Assert: 仅包含 diff，不包含 stat 和 file 列表。
	if result != diff {
		t.Errorf("Small 分级应仅返回原始 diff\n  实际: %q\n  期望: %q", result, diff)
	}
	if strings.Contains(result, "变更统计") {
		t.Error("Small 分级不应包含 stat")
	}
}

// ============================================================================
// TC-T2.3-02: Medium 分级 → 完整 diff + --stat
// ============================================================================
func TestBuildContext_TC_T2_3_02_Medium_DiffAndStat(t *testing.T) {
	// Arrange
	diff := "--- a/main.go\n+++ b/main.go\n@@ -1,5 +1,6 @@"
	stat := "main.go | 6 +++---"
	fileNames := []string{"main.go"}

	// Act
	result := BuildContext(diff, stat, fileNames, Medium)

	// Assert: 包含 stat 和 diff。
	if !strings.Contains(result, "变更统计") {
		t.Error("Medium 分级应包含 stat 部分")
	}
	if !strings.Contains(result, stat) {
		t.Error("Medium 分级应包含 stat 内容")
	}
	if !strings.Contains(result, "变更详情") {
		t.Error("Medium 分级应包含 diff 详情")
	}
	if !strings.Contains(result, diff) {
		t.Error("Medium 分级应包含 diff 内容")
	}
}

// ============================================================================
// TC-T2.3-03: Large 分级 → 摘要压缩（不含原始完整 diff）
// ============================================================================
func TestBuildContext_TC_T2_3_03_Large_SummaryOnly(t *testing.T) {
	// Arrange: 模拟一个超过 50 行的文件 diff。
	diff := `diff --git a/big.go b/big.go
index abc..def 100644
--- a/big.go
+++ b/big.go
@@ -1,50 +1,50 @@
-old content line 1
-old content line 2
-old content line 3
-old content line 4
-old content line 5
-old content line 6
-old content line 7
-old content line 8
-old content line 9
-old content line 10
-old content line 11
-old content line 12
-old content line 13
-old content line 14
-old content line 15
-old content line 16
-old content line 17
-old content line 18
-old content line 19
-old content line 20
-old content line 21
-old content line 22
-old content line 23
-old content line 24
-old content line 25
-old content line 26
-old content line 27
-old content line 28
-old content line 29
-old content line 30
-old content line 31
-old content line 32
-old content line 33
-old content line 34
-old content line 35
-old content line 36
-old content line 37
-old content line 38
-old content line 39
-old content line 40
-old content line 41
-old content line 42
-old content line 43
-old content line 44
-old content line 45
-old content line 46
-old content line 47
-old content line 48
-old content line 49
-old content line 50
+new content line 1
+new content line 2
+new content line 3
+new content line 4
+new content line 5
+new content line 6
+new content line 7
+new content line 8
+new content line 9
+new content line 10
+new content line 11
+new content line 12
+new content line 13
+new content line 14
+new content line 15
+new content line 16
+new content line 17
+new content line 18
+new content line 19
+new content line 20
+new content line 21
+new content line 22
+new content line 23
+new content line 24
+new content line 25
+new content line 26
+new content line 27
+new content line 28
+new content line 29
+new content line 30
+new content line 31
+new content line 32
+new content line 33
+new content line 34
+new content line 35
+new content line 36
+new content line 37
+new content line 38
+new content line 39
+new content line 40
+new content line 41
+new content line 42
+new content line 43
+new content line 44
+new content line 45
+new content line 46
+new content line 47
+new content line 48
+new content line 49
+new content line 50
`
	stat := "big.go | 100 +-"
	fileNames := []string{"big.go"}

	// Act
	result := BuildContext(diff, stat, fileNames, Large)

	// Assert: Large 应输出摘要压缩，包含文件路径和行数统计。
	if !strings.Contains(result, "big.go") {
		t.Error("Large 摘要应包含文件路径")
	}
	if !strings.Contains(result, "行") {
		t.Error("Large 摘要应包含行数统计")
	}
	// 不应包含原始 diff 中的具体内容行（摘要已压缩）。
	if strings.Contains(result, "old content line 1") {
		t.Error("Large 摘要不应包含原始 diff 的具体内容行")
	}
}

// ============================================================================
// TC-T2.3-04: ExtraLarge 分级 → --stat + --name-only + 拆分提示
// ============================================================================
func TestBuildContext_TC_T2_3_04_ExtraLarge_StatAndFileList(t *testing.T) {
	// Arrange
	diff := "very large diff..."
	stat := "a.go | 5000 +-\nb.go | 6000 +-\n"
	fileNames := []string{"a.go", "b.go"}

	// Act
	result := BuildContext(diff, stat, fileNames, ExtraLarge)

	// Assert: 应包含 stat、文件列表和拆分建议。
	if !strings.Contains(result, "变更统计") {
		t.Error("ExtraLarge 应包含 stat 部分")
	}
	if !strings.Contains(result, "a.go") {
		t.Error("ExtraLarge 应包含文件列表")
	}
	if !strings.Contains(result, "建议拆分") {
		t.Error("ExtraLarge 应包含拆分建议")
	}
	// 不应包含完整 diff。
	if strings.Contains(result, "very large diff") {
		t.Error("ExtraLarge 不应包含完整 diff")
	}
}
