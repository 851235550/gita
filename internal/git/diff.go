// Package git 封装 gita 所需的 Git 命令调用。
// 所有能力基于 git diff / git status 等 Git 原生命令，
// 而非直接分析源码目录，保证语义聚焦在"变更"本身（需求文档 4.1 节）。
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetStagedDiff 返回 git diff --staged 的完整输出。
// 仅在已通过 HasStagedChanges() 确认有暂存变更后调用，
// 否则返回空字符串而非报错。
func GetStagedDiff() (string, error) {
	return runGitCmd("diff", "--staged")
}

// GetStagedStat 返回 git diff --staged --stat 的统计摘要，
// 包含变更文件列表与增删行数概要。
func GetStagedStat() (string, error) {
	return runGitCmd("diff", "--staged", "--stat")
}

// GetStagedFileNames 返回所有暂存变更文件的路径列表（不含统计信息）。
// 底层使用 git diff --staged --name-only。
func GetStagedFileNames() ([]string, error) {
	out, err := runGitCmd("diff", "--staged", "--name-only")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(out), "\n"), nil
}

// IsGitRepo 判断当前目录是否位于有效的 Git 仓库中（包括子目录）。
// 通过 git rev-parse --is-inside-work-tree 实现，
// 比检查 .git 目录更可靠（兼容 worktree、submodule 等场景）。
func IsGitRepo() bool {
	_, err := runGitCmd("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// HasStagedChanges 判断暂存区是否有变更。
// 使用 git diff --staged --quiet 检测，该命令在无变更时退出码为 0。
func HasStagedChanges() bool {
	cmd := exec.Command("git", "diff", "--staged", "--quiet")
	err := cmd.Run()
	// --quiet: 无差异时退出码 0，有差异时退出码 1，出错时退出码 >1。
	return err != nil
}

// runGitCmd 执行 git 命令并返回 stdout 字符串。
// stderr 内容会附加到 error 中以辅助排查。
func runGitCmd(args ...string) (string, error) {
	cmd := exec.Command("git", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// 将 git 自身的 stderr 附加到错误信息中，便于定位问题（如"不是 git 仓库"）。
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("git %s 失败: %s", strings.Join(args, " "), strings.TrimSpace(errMsg))
	}

	return stdout.String(), nil
}
