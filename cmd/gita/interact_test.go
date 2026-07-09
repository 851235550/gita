package main

import (
	"bytes"
	"strings"
	"testing"
)

// ============================================================================
// TC-T4.1-01: 用户直接回车（默认 Y）→ 确认提交
// ============================================================================
func TestConfirm_TC_T4_1_01_DefaultEnter_Confirms(t *testing.T) {
	// Arrange: 用户直接回车（空输入 = Y）。
	in := strings.NewReader("\n")
	var out bytes.Buffer
	c := &interactConfirmer{in: in, out: &out}

	// Act
	result, msg, err := c.confirm("feat(test): test commit")

	// Assert
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if result != interactConfirm {
		t.Errorf("result = %v, want interactConfirm", result)
	}
	if msg != "feat(test): test commit" {
		t.Errorf("msg = %q, want original message", msg)
	}
}

// ============================================================================
// TC-T4.1-02: 用户输入 n → 取消
// ============================================================================
func TestConfirm_TC_T4_1_02_InputN_Cancels(t *testing.T) {
	// Arrange
	in := strings.NewReader("n\n")
	var out bytes.Buffer
	c := &interactConfirmer{in: in, out: &out}

	// Act
	result, msg, err := c.confirm("test message")

	// Assert
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if result != interactCancel {
		t.Errorf("result = %v, want interactCancel", result)
	}
	if msg != "" {
		t.Errorf("取消时 msg 应为空，实际: %q", msg)
	}
}

// ============================================================================
// TC-T4.1-03: 用户输入 r → 重新生成
// ============================================================================
func TestConfirm_TC_T4_1_03_InputR_Regenerates(t *testing.T) {
	// Arrange
	in := strings.NewReader("r\n")
	var out bytes.Buffer
	c := &interactConfirmer{in: in, out: &out}

	// Act
	result, _, err := c.confirm("test message")

	// Assert
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if result != interactRegenerate {
		t.Errorf("result = %v, want interactRegenerate", result)
	}
}

// ============================================================================
// TC-T4.1-07: 用户输入非法字符 x → 重新提示
// ============================================================================
func TestConfirm_TC_T4_1_07_InvalidInput_Reprompts(t *testing.T) {
	// Arrange: 先输入非法字符 x，再输入 y。
	in := strings.NewReader("x\ny\n")
	var out bytes.Buffer
	c := &interactConfirmer{in: in, out: &out}

	// Act
	result, _, err := c.confirm("test message")

	// Assert
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if result != interactConfirm {
		t.Errorf("最终应确认为 confirm，实际: %v", result)
	}
	// 应包含提示重新输入的文案。
	if !strings.Contains(out.String(), "请输入 Y/e/r/n") {
		t.Error("非法输入后应提示 '请输入 Y/e/r/n'")
	}
}

// ============================================================================
// 补充: 多次 r 输入（TC-T4.1-06）
// ============================================================================
func TestConfirm_MultipleR_EachReturnsRegenerate(t *testing.T) {
	// Arrange: 每次 confirm 调用使用独立的 Reader，避免 bufio 缓冲干扰。
	for i := 0; i < 5; i++ {
		in := strings.NewReader("r\n")
		var out bytes.Buffer
		c := &interactConfirmer{in: in, out: &out}

		// Act
		result, _, err := c.confirm("msg")

		// Assert
		if err != nil {
			t.Fatalf("第 %d 次不应返回错误: %v", i+1, err)
		}
		if result != interactRegenerate {
			t.Errorf("第 %d 次 result = %v, want interactRegenerate", i+1, result)
		}
	}

	// 最终取消。
	in := strings.NewReader("n\n")
	var out bytes.Buffer
	c := &interactConfirmer{in: in, out: &out}
	result, _, err := c.confirm("msg")
	if err != nil {
		t.Fatalf("最终取消不应报错: %v", err)
	}
	if result != interactCancel {
		t.Errorf("最终 result = %v, want interactCancel", result)
	}
}

// ============================================================================
// 补充: Y 大写输入
// ============================================================================
func TestConfirm_InputY_Confirms(t *testing.T) {
	// Arrange
	in := strings.NewReader("Y\n")
	var out bytes.Buffer
	c := &interactConfirmer{in: in, out: &out}

	// Act
	result, _, err := c.confirm("msg")

	// Assert
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if result != interactConfirm {
		t.Errorf("result = %v, want interactConfirm", result)
	}
}

// ============================================================================
// 补充: yes 输入
// ============================================================================
func TestConfirm_InputYes_Confirms(t *testing.T) {
	// Arrange
	in := strings.NewReader("yes\n")
	var out bytes.Buffer
	c := &interactConfirmer{in: in, out: &out}

	// Act
	result, _, err := c.confirm("msg")

	// Assert
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if result != interactConfirm {
		t.Errorf("result = %v, want interactConfirm", result)
	}
}
