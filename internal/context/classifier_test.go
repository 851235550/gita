package context

import (
	"testing"
)

// ============================================================================
// TC-T2.1-01: 0 行 → Small
// ============================================================================
func TestClassifyDiff_TC_T2_1_01_ZeroLines_Small(t *testing.T) {
	// Arrange: 空 diff（无变更）。
	lineCount := 0
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert
	if level != Small {
		t.Errorf("0 行应判定为 Small，实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-02: 999 行 → Small（< 1000 边界内）
// ============================================================================
func TestClassifyDiff_TC_T2_1_02_999Lines_Small(t *testing.T) {
	// Arrange
	lineCount := 999
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert: 999 < 1000，仍在 Small 范围内。
	if level != Small {
		t.Errorf("999 行应判定为 Small，实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-03: 1000 行 → Medium（含边界，左闭）
// ============================================================================
func TestClassifyDiff_TC_T2_1_03_1000Lines_Medium(t *testing.T) {
	// Arrange: 1000 是 Small/Medium 边界，按左闭约定 1000 ∈ Medium。
	lineCount := 1000
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert
	if level != Medium {
		t.Errorf("1000 行应判定为 Medium（边界左闭），实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-04: 4999 行 → Medium（< maxDiffLines 边界内）
// ============================================================================
func TestClassifyDiff_TC_T2_1_04_4999Lines_Medium(t *testing.T) {
	// Arrange
	lineCount := 4999
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert: 4999 < 5000，仍在 Medium 范围内。
	if level != Medium {
		t.Errorf("4999 行应判定为 Medium，实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-05: 5000 行 → Large（含边界，默认 maxDiffLines = 5000）
// ============================================================================
func TestClassifyDiff_TC_T2_1_05_5000Lines_Large(t *testing.T) {
	// Arrange: 5000 是 Medium/Large 边界，按左闭约定 5000 ∈ Large。
	lineCount := 5000
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert
	if level != Large {
		t.Errorf("5000 行应判定为 Large（边界左闭），实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-06: 14999 行 → Large（< 15000 边界内）
// ============================================================================
func TestClassifyDiff_TC_T2_1_06_14999Lines_Large(t *testing.T) {
	// Arrange
	lineCount := 14999
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert
	if level != Large {
		t.Errorf("14999 行应判定为 Large，实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-07: 15000 行 → ExtraLarge（含边界，左闭）
// ============================================================================
func TestClassifyDiff_TC_T2_1_07_15000Lines_ExtraLarge(t *testing.T) {
	// Arrange: 15000 是 Large/ExtraLarge 边界，按左闭约定 15000 ∈ ExtraLarge。
	lineCount := 15000
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert
	if level != ExtraLarge {
		t.Errorf("15000 行应判定为 ExtraLarge（边界左闭），实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-08: 15001 行 → ExtraLarge
// ============================================================================
func TestClassifyDiff_TC_T2_1_08_15001Lines_ExtraLarge(t *testing.T) {
	// Arrange
	lineCount := 15001
	maxDiffLines := 5000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert
	if level != ExtraLarge {
		t.Errorf("15001 行应判定为 ExtraLarge，实际: %v", level)
	}
}

// ============================================================================
// TC-T2.1-09: 用户将 max_diff_lines 改为 2000，1500 行 → Medium
// ============================================================================
func TestClassifyDiff_TC_T2_1_09_CustomMaxDiffLines(t *testing.T) {
	// Arrange: 用户修改配置 max_diff_lines = 2000，1500 行应变为 Medium（而非默认配置下的 Large）。
	lineCount := 1500
	maxDiffLines := 2000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert: 1500 < 2000，仍在 Medium 范围。
	if level != Medium {
		t.Errorf("maxDiffLines=2000 时 1500 行应判定为 Medium，实际: %v", level)
	}
}

// ============================================================================
// 补充: maxDiffLines 为异常值 0 → 回退默认 5000
// ============================================================================
func TestClassifyDiff_ZeroMaxDiffLines_FallbackToDefault(t *testing.T) {
	// Arrange: 异常配置，maxDiffLines = 0。
	lineCount := 3000
	maxDiffLines := 0

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert: 回退到默认 5000，3000 < 5000 → Medium。
	if level != Medium {
		t.Errorf("maxDiffLines=0 时应回退默认 5000，3000 行应为 Medium，实际: %v", level)
	}
}

// ============================================================================
// 补充: maxDiffLines 为负数 → 回退默认 5000
// ============================================================================
func TestClassifyDiff_NegativeMaxDiffLines_FallbackToDefault(t *testing.T) {
	// Arrange
	lineCount := 6000
	maxDiffLines := -100

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert: 回退默认 5000，6000 >= 5000 且 < 15000 → Large。
	if level != Large {
		t.Errorf("maxDiffLines 为负数时应回退默认 5000，6000 行应为 Large，实际: %v", level)
	}
}

// ============================================================================
// 补充: maxDiffLines 被设置为远超 15000 的值
// ============================================================================
func TestClassifyDiff_MaxDiffLinesExceedsExtraLarge(t *testing.T) {
	// Arrange: maxDiffLines = 20000，大于 extraLargeThreshold。
	lineCount := 18000
	maxDiffLines := 20000

	// Act
	level := ClassifyDiff(lineCount, maxDiffLines)

	// Assert: 18000 >= 15000 → ExtraLarge，不受 maxDiffLines 影响。
	if level != ExtraLarge {
		t.Errorf("18000 行应判定为 ExtraLarge（不受 maxDiffLines 影响），实际: %v", level)
	}
}
