package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo 在指定目录下初始化一个临时 Git 仓库，并配置 user.name/user.email 以避免提交报错。
// 返回仓库根目录路径。
func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	runInDir(t, dir, "git", "init")
	// 设置最小 git 身份信息，避免后续 commit/add 报错。
	runInDir(t, dir, "git", "config", "user.name", "test")
	runInDir(t, dir, "git", "config", "user.email", "test@test.com")
}

// runInDir 在指定目录下执行命令，若失败则标记测试为 Fatal。
func runInDir(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("命令 %s %s 失败: %v\n输出: %s", name, strings.Join(args, " "), err, string(out))
	}
}

// writeFile 在指定目录下写入文件内容。
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("写入文件 %s 失败: %v", name, err)
	}
}

// ============================================================================
// TC-T1.1-01: 有效 Git 仓库，无 staged 变更 → HasStagedChanges() 返回 false
// ============================================================================
func TestHasStagedChanges_NoChanges_ReturnsFalse(t *testing.T) {
	// Arrange: 初始化空 Git 仓库，不创建任何文件。
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Act: 切换到仓库目录后检测。
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	result := HasStagedChanges()

	// Assert
	if result {
		t.Error("无暂存变更时 HasStagedChanges() 应返回 false")
	}
}

// ============================================================================
// TC-T1.1-02: git add 一个新文件后 → HasStagedChanges() 返回 true
// ============================================================================
func TestHasStagedChanges_AfterAdd_ReturnsTrue(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	writeFile(t, tmpDir, "hello.txt", "hello world")
	// 需要先做一次初始提交，否则 git diff --staged 在无 HEAD 时行为不同。
	runInDir(t, tmpDir, "git", "add", "hello.txt")
	runInDir(t, tmpDir, "git", "commit", "-m", "init")

	// 创建新的变更并 stage。
	writeFile(t, tmpDir, "hello.txt", "hello world v2")
	runInDir(t, tmpDir, "git", "add", "hello.txt")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Act
	result := HasStagedChanges()

	// Assert
	if !result {
		t.Error("git add 后 HasStagedChanges() 应返回 true")
	}
}

// ============================================================================
// TC-T1.1-03: 非 Git 仓库 → IsGitRepo() 返回 false，不 panic
// ============================================================================
func TestIsGitRepo_NonGitDir_ReturnsFalse(t *testing.T) {
	// Arrange: 临时目录未初始化 Git。
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Act
	result := IsGitRepo()

	// Assert
	if result {
		t.Error("非 Git 仓库中 IsGitRepo() 应返回 false")
	}
}

// ============================================================================
// TC-T1.1-04: 非 Git 仓库 → GetStagedDiff() 返回 error
// ============================================================================
func TestGetStagedDiff_NonGitRepo_ReturnsError(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Act
	_, err := GetStagedDiff()

	// Assert
	if err == nil {
		t.Error("非 Git 仓库中 GetStagedDiff() 应返回 error")
	}
	// 错误信息应包含关键提示。
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("错误信息应包含 'git' 关键字，实际: %v", err)
	}
}

// ============================================================================
// TC-T1.1-05: staged 一个修改了内容的文件 → diff 包含 unified diff 格式
// ============================================================================
func TestGetStagedDiff_ModifiedFile_ContainsUnifiedDiff(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	writeFile(t, tmpDir, "main.go", "package main\n\nfunc main() {\n}\n")
	runInDir(t, tmpDir, "git", "add", "main.go")
	runInDir(t, tmpDir, "git", "commit", "-m", "init")

	// 修改文件并 stage。
	writeFile(t, tmpDir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")
	runInDir(t, tmpDir, "git", "add", "main.go")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Act
	diff, err := GetStagedDiff()

	// Assert
	if err != nil {
		t.Fatalf("GetStagedDiff() 返回错误: %v", err)
	}
	if !strings.Contains(diff, "diff --git") {
		t.Error("diff 输出应包含 'diff --git' 标记")
	}
	if !strings.Contains(diff, "@@") {
		t.Error("diff 输出应包含 '@@' 行号标记")
	}
}

// ============================================================================
// TC-T1.1-06: staged 一个二进制文件 → 显示 "Binary files differ"，不崩溃
// ============================================================================
func TestGetStagedDiff_BinaryFile_NoCrash(t *testing.T) {
	// Arrange: 创建一个简单二进制文件（最小 PNG header）。
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	writeFile(t, tmpDir, "image.png", "\x89PNG\r\n\x1a\n")
	runInDir(t, tmpDir, "git", "add", "image.png")
	runInDir(t, tmpDir, "git", "commit", "-m", "init")

	// 修改二进制文件并 stage。
	writeFile(t, tmpDir, "image.png", "\x89PNG\r\n\x1a\n\x00\x00\x00")
	runInDir(t, tmpDir, "git", "add", "image.png")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Act
	diff, err := GetStagedDiff()

	// Assert: 不应崩溃，Git 对二进制文件的展示为 "Binary files differ"。
	if err != nil {
		t.Fatalf("二进制文件 diff 不应报错: %v", err)
	}
	if !strings.Contains(diff, "Binary files") {
		t.Errorf("二进制文件 diff 应包含 'Binary files' 说明，实际输出: %s", diff)
	}
}

// ============================================================================
// TC-T1.1-07: staged 3 个文件 → GetStagedFileNames() 返回 3 个路径
// ============================================================================
func TestGetStagedFileNames_ThreeFiles_ReturnsThree(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	writeFile(t, tmpDir, "a.go", "package a")
	writeFile(t, tmpDir, "b.go", "package b")
	writeFile(t, tmpDir, "c.go", "package c")
	runInDir(t, tmpDir, "git", "add", ".")
	runInDir(t, tmpDir, "git", "commit", "-m", "init")

	// 修改三个文件并全部 stage。
	writeFile(t, tmpDir, "a.go", "package a\n// v2")
	writeFile(t, tmpDir, "b.go", "package b\n// v2")
	writeFile(t, tmpDir, "c.go", "package c\n// v2")
	runInDir(t, tmpDir, "git", "add", ".")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Act
	files, err := GetStagedFileNames()

	// Assert
	if err != nil {
		t.Fatalf("GetStagedFileNames() 返回错误: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("文件数 = %d, want 3", len(files))
	}
}

// ============================================================================
// TC-T1.1: GetStagedStat 正常返回统计信息
// ============================================================================
func TestGetStagedStat_ReturnsStat(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)
	writeFile(t, tmpDir, "main.go", "package main\n\nfunc main() {\n}\n")
	runInDir(t, tmpDir, "git", "add", "main.go")
	runInDir(t, tmpDir, "git", "commit", "-m", "init")

	writeFile(t, tmpDir, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n")
	runInDir(t, tmpDir, "git", "add", "main.go")

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	// Act
	stat, err := GetStagedStat()

	// Assert
	if err != nil {
		t.Fatalf("GetStagedStat() 返回错误: %v", err)
	}
	if !strings.Contains(stat, "main.go") {
		t.Errorf("stat 输出应包含文件名, 实际: %s", stat)
	}
}
