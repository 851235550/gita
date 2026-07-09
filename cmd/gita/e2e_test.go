package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// e2eGitInit 在临时目录初始化 Git 仓库并返回目录路径。
func e2eGitInit(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.name", "e2e-test")
	runCmd(t, dir, "git", "config", "user.email", "e2e@test.com")
	return dir
}

// e2eWriteFile 在仓库中写入文件并 git add + commit 初始版本。
func e2eWriteAndStage(t *testing.T, dir, name, content string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写入文件失败: %v", err)
	}
	runCmd(t, dir, "git", "add", name)
}

// e2eWriteLargeFile 生成指定行数的文件并 stage。
func e2eWriteLargeFile(t *testing.T, dir, name string, lines int, baseContent string) {
	t.Helper()

	var sb strings.Builder
	for i := 0; i < lines; i++ {
		sb.WriteString(fmt.Sprintf("%s line %d\n", baseContent, i+1))
	}
	e2eWriteAndStage(t, dir, name, sb.String())
}

// runCmd 在指定目录执行命令。
func runCmd(t *testing.T, dir, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("命令 %s %s 失败: %v\n输出: %s", name, strings.Join(args, " "), err, string(out))
	}
}

// ============================================================================
// TC-E2E-01: 完整流程 —— 初始化仓库 → 创建文件 → git add → gita commit → 确认
// ============================================================================
func TestE2E_TC_E2E_01_FullFlow(t *testing.T) {
	// Arrange
	dir := e2eGitInit(t)
	e2eWriteAndStage(t, dir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

	// 使用 mock provider（默认 MockProvider 返回预设文本）。
	mock := &mockProviderForE2E{}

	// 模拟用户输入：直接回车确认。
	in := strings.NewReader("\n")
	var out bytes.Buffer

	// Act: 构造并执行 commit 流程。
	err := e2eRunCommit(in, &out, mock, dir, "--lang", "zh-CN")

	// Assert
	if err != nil {
		t.Fatalf("e2e commit 失败: %v", err)
	}

	// 验证 git log 中出现了新 commit。
	logOut := runCmdCapture(t, dir, "git", "log", "--oneline", "-1")
	if logOut == "" {
		t.Error("应产生一条新 commit")
	}
}

// ============================================================================
// TC-E2E-05: --hint 参数 → prompt 中包含 hint 文本
// ============================================================================
func TestE2E_TC_E2E_05_HintInPrompt(t *testing.T) {
	// Arrange
	dir := e2eGitInit(t)
	e2eWriteAndStage(t, dir, "fix.go", "package main\n// hotfix\n")

	mock := &mockProviderForE2E{
		capturePrompt: true,
	}

	in := strings.NewReader("\n")
	var out bytes.Buffer

	// Act
	err := e2eRunCommit(in, &out, mock, dir, "--hint", "这是一个紧急 hotfix")

	// Assert
	if err != nil {
		t.Fatalf("e2e commit 失败: %v", err)
	}
	if !strings.Contains(mock.lastPrompt, "这是一个紧急 hotfix") {
		t.Errorf("prompt 应包含 --hint 文本\n实际 prompt: %s", mock.lastPrompt)
	}
}

// ============================================================================
// TC-E2E-03: 超大 diff 不传 --force → 阻塞确认
// ============================================================================
func TestE2E_TC_E2E_03_ExtraLargeNoForce_Blocks(t *testing.T) {
	// Arrange: 创建一个超过 15000 行的 diff。
	dir := e2eGitInit(t)
	e2eWriteAndStage(t, dir, "big.go", "package big\n")
	runCmd(t, dir, "git", "commit", "-m", "init")

	// 生成大量行数的变更。
	var sb strings.Builder
	for i := 0; i < 15100; i++ {
		sb.WriteString(fmt.Sprintf("var x%d = %d\n", i, i))
	}
	e2eWriteAndStage(t, dir, "big.go", sb.String())

	mock := &mockProviderForE2E{}

	// 用户输入 n（拒绝继续）。
	in := strings.NewReader("n\n")
	var out bytes.Buffer

	// Act
	err := e2eRunCommit(in, &out, mock, dir)

	// Assert: 应因用户拒绝而取消。
	if err == nil {
		t.Fatal("超大 diff 且用户拒绝，应返回错误")
	}
	if !strings.Contains(err.Error(), "取消") {
		t.Errorf("错误应包含'取消': %v", err)
	}
}

// ============================================================================
// TC-E2E-04: 超大 diff 传 --force → 跳过确认直接生成
// ============================================================================
func TestE2E_TC_E2E_04_ExtraLargeWithForce_Proceeds(t *testing.T) {
	// Arrange
	dir := e2eGitInit(t)
	e2eWriteAndStage(t, dir, "big.go", "package big\n")
	runCmd(t, dir, "git", "commit", "-m", "init")

	var sb strings.Builder
	for i := 0; i < 15100; i++ {
		sb.WriteString(fmt.Sprintf("var x%d = %d\n", i, i))
	}
	e2eWriteAndStage(t, dir, "big.go", sb.String())

	mock := &mockProviderForE2E{}

	in := strings.NewReader("\n")
	var out bytes.Buffer

	// Act: 传入 --force。
	err := e2eRunCommit(in, &out, mock, dir, "--force")

	// Assert: --force 跳过确认，直接生成。
	if err != nil {
		t.Fatalf("--force 不应阻塞: %v", err)
	}
}

// ============================================================================
// TC-E2E-07: 未设置 API Key → 非零退出码
// ============================================================================
func TestE2E_TC_E2E_07_NoAPIKey_Error(t *testing.T) {
	// Arrange
	dir := e2eGitInit(t)
	e2eWriteAndStage(t, dir, "main.go", "package main\n")

	// 清除可能的 API Key 环境变量，使用真实 provider 以触发 API Key 检查。
	t.Setenv("GITA_DEEPSEEK_API_KEY", "")

	in := strings.NewReader("\n")
	var out bytes.Buffer

	// Act: 不使用 mock，触发真实的 API Key 检查（但不会真正调用 LLM）。
	err := e2eRunCommitRealProvider(in, &out, dir)

	// Assert: 应因 API Key 缺失而报错。
	if err == nil {
		t.Fatal("未设置 API Key 应返回错误")
	}
	if !strings.Contains(err.Error(), "GITA_DEEPSEEK_API_KEY") {
		t.Errorf("错误应包含环境变量名: %v", err)
	}
}

// runCmdCapture 在指定目录执行命令并返回 stdout 字符串。
func runCmdCapture(t *testing.T, dir, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("命令 %s %s 失败: %v", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
}
