package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// interactResult 表示用户对生成结果的确认操作。
type interactResult int

const (
	interactConfirm    interactResult = iota // Y: 确认提交
	interactEdit                             // e: 编辑后提交
	interactRegenerate                       // r: 重新生成
	interactCancel                           // n: 取消
)

// interactConfirmer 封装用户交互的输入输出通道，便于测试时注入 mock。
type interactConfirmer struct {
	in  io.Reader // 标准输入（测试时可替换为 strings.Reader）。
	out io.Writer // 标准输出（测试时可替换为 bytes.Buffer）。
}

// newInteractConfirmer 创建使用标准输入输出的交互器。
// 定义为函数变量以支持测试时注入自定义 I/O 通道。
var newInteractConfirmer = func() *interactConfirmer {
	return &interactConfirmer{
		in:  os.Stdin,
		out: os.Stdout,
	}
}

// confirm 展示建议的 commit message 并等待用户输入。
// 返回用户选择的操作和（若为编辑模式）编辑后的文本。
//
// 注意：此处刻意不限制重新生成次数。
// Gita 是开源工具，用户使用自己的 LLM Key，调用成本由用户自行承担，
// 详见需求文档 5.3 节 Non-goals。
func (c *interactConfirmer) confirm(commitMsg string) (interactResult, string, error) {
	c.printMessage(commitMsg)

	reader := bufio.NewReader(c.in)

	for {
		fmt.Fprint(c.out, "\n[Y] 确认提交  [e] 编辑后提交  [r] 重新生成  [n] 取消\n> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return interactCancel, "", fmt.Errorf("读取用户输入失败: %w", err)
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "", "y", "yes":
			return interactConfirm, commitMsg, nil

		case "n", "no":
			return interactCancel, "", nil

		case "r":
			return interactRegenerate, "", nil

		case "e":
			edited, err := c.openEditor(commitMsg)
			if err != nil {
				return interactCancel, "", err
			}
			return interactEdit, edited, nil

		default:
			fmt.Fprintf(c.out, "请输入 Y/e/r/n\n")
		}
	}
}

// printMessage 展示建议的 Commit Message。
func (c *interactConfirmer) printMessage(msg string) {
	fmt.Fprintln(c.out, "\nSuggested Commit Message")
	fmt.Fprintln(c.out, "─────────────────────────")
	fmt.Fprintln(c.out, msg)
	fmt.Fprintln(c.out, "─────────────────────────")
}

// openEditor 打开 $EDITOR 供用户编辑 commit message。
// 若 $EDITOR 未设置，降级为内置简单输入（读取一行 stdin）。
// 降级而非报错，因为 $EDITOR 不是 Gita 的强制依赖。
func (c *interactConfirmer) openEditor(msg string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// 降级：$EDITOR 未设置时，使用内置简单输入。
		fmt.Fprintln(c.out, "$EDITOR 未设置，请直接输入编辑后的 Commit Message（回车提交）:")
		reader := bufio.NewReader(c.in)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("读取编辑输入失败: %w", err)
		}
		return strings.TrimSpace(input), nil
	}

	// 将当前消息写入临时文件供编辑器修改。
	tmpFile, err := os.CreateTemp("", "gita-commit-*.txt")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(msg); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmpFile.Close()

	// 启动编辑器，阻塞等待用户完成编辑。
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("编辑器 %s 执行失败: %w", editor, err)
	}

	// 读取编辑后的内容。
	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("读取编辑结果失败: %w", err)
	}

	result := strings.TrimSpace(string(edited))
	if result == "" {
		return "", fmt.Errorf("编辑后的 Commit Message 不能为空")
	}

	return result, nil
}

// execGitCommit 执行 git commit -m "message"。
func execGitCommit(msg string) error {
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
