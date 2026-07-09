package main

import (
	"context"
	"io"
	"os"

	"gita/internal/llm"
)

// mockProviderForE2E 是用于 e2e 测试的 LLM Provider mock。
// 支持捕获发送给 LLM 的 prompt 内容，便于断言。
type mockProviderForE2E struct {
	// capturePrompt 为 true 时，每次 Generate 调用会将 prompt 保存到 lastPrompt。
	capturePrompt bool

	// lastPrompt 保存最后一次调用的 prompt 内容。
	lastPrompt string

	// callCount 记录调用次数。
	callCount int
}

// Generate 实现 llm.Provider 接口，返回预设的 commit message。
func (m *mockProviderForE2E) Generate(ctx context.Context, prompt string) (string, error) {
	m.callCount++
	if m.capturePrompt {
		m.lastPrompt = prompt
	}
	return "feat(e2e): mock generated commit for testing", nil
}

// e2eRunCommit 在测试环境中运行 gita commit 流程。
// 使用 mock provider 替代真实 LLM，通过 in/out 控制交互输入输出。
// args 为 commit 子命令的额外参数（如 --hint、--force）。
// workDir 为 Git 仓库的工作目录。
func e2eRunCommit(in io.Reader, out io.Writer, mock llm.Provider, workDir string, args ...string) error {
	// 注入 mock provider 和测试输入。
	testProvider = mock
	testStdin = in
	defer func() {
		testProvider = nil
		testStdin = nil
	}()

	// 切换到临时仓库目录执行（测试结束后恢复）。
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		return err
	}
	defer os.Chdir(origDir)

	// 构建完整的命令行参数。
	fullArgs := append([]string{"commit"}, args...)
	flags, err := parseCommitFlags(fullArgs[1:])
	if err != nil {
		return err
	}

	// 直接调用 runCommit，注入 mock I/O。
	return runCommitWithIO(flags, in, out)
}

// e2eRunCommitRealProvider 使用真实 Provider（不注入 mock），用于测试 API Key 缺失等场景。
func e2eRunCommitRealProvider(in io.Reader, out io.Writer, workDir string) error {
	flags := &commitFlags{}

	// 确保不使用 mock，注入测试输入。
	testProvider = nil
	testStdin = in
	defer func() { testStdin = nil }()

	// 切换到临时仓库目录。
	origDir, _ := os.Getwd()
	if err := os.Chdir(workDir); err != nil {
		return err
	}
	defer os.Chdir(origDir)

	return runCommitWithIO(flags, in, out)
}

// runCommitWithIO 是 runCommit 的测试友好版本，允许注入 I/O。
// 在测试中替换 confirmer 的 in/out，避免依赖真实终端 I/O。
func runCommitWithIO(flags *commitFlags, in io.Reader, out io.Writer) error {
	// 临时替换 confirmer 工厂函数，注入测试 I/O 通道。
	origNewConfirmer := newInteractConfirmer
	defer func() { newInteractConfirmer = origNewConfirmer }()

	newInteractConfirmer = func() *interactConfirmer {
		return &interactConfirmer{in: in, out: out}
	}

	return runCommit(flags)
}
