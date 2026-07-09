// Gita —— 基于 Git 与大语言模型（LLM）的开发辅助 CLI 工具。
// MVP 阶段提供 gita commit 命令，基于 staged diff 自动生成 Commit Message。
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"gita/internal/config"
	gcontext "gita/internal/context"
	"gita/internal/git"
	"gita/internal/llm"
	"gita/internal/prompt"
)

// testProvider 用于测试时注入 mock LLM Provider，绕过真实 API 调用。
// 仅在测试中设置，生产代码中始终为 nil。
var testProvider llm.Provider

// testStdin 用于测试时注入标准输入，替代 os.Stdin。
// 仅在测试中设置，生产代码中始终为 nil。
var testStdin io.Reader

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "gita: %v\n", err)
		os.Exit(1)
	}
}

// run 解析并执行子命令，是 main 的业务入口，便于测试时直接传入参数。
func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "commit":
		flags, err := parseCommitFlags(args[1:])
		if err != nil {
			// --help 或 -h 属于正常请求，打印帮助后正常退出，不视为错误。
			if err == flag.ErrHelp {
				printCommitHelp()
				return nil
			}
			return fmt.Errorf("参数错误: %w", err)
		}
		return runCommit(flags)
	case "help", "--help", "-h":
		printHelp()
	default:
		return fmt.Errorf("未知命令: %s，运行 'gita --help' 查看可用命令", args[0])
	}
	return nil
}

// runCommit 执行 gita commit 主流程：
// 读取 staged changes → 分级 → 构造上下文 → 调用 LLM → 输出结果。
// 当前为 T3.1 骨架版本，暂不含交互确认（Y/e/r/n），
// 交互确认在 T4.1 实现。
func runCommit(flags *commitFlags) error {
	// 1. 检查是否在 Git 仓库中。
	if !git.IsGitRepo() {
		return fmt.Errorf("当前目录不是有效的 Git 仓库")
	}

	// 2. 检查是否有暂存变更。
	if !git.HasStagedChanges() {
		return fmt.Errorf("没有检测到暂存变更，请先执行 git add")
	}

	// 3. 加载配置。
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 4. 获取 Git 数据。
	diff, err := git.GetStagedDiff()
	if err != nil {
		return fmt.Errorf("获取 diff 失败: %w", err)
	}

	stat, err := git.GetStagedStat()
	if err != nil {
		return fmt.Errorf("获取 stat 失败: %w", err)
	}

	fileNames, err := git.GetStagedFileNames()
	if err != nil {
		return fmt.Errorf("获取文件列表失败: %w", err)
	}

	// 5. 分级并构造上下文。
	// 注意：diff 行数计数基于换行符数量，与需求文档 6.4 节一致。
	lineCount := strings.Count(diff, "\n")
	level := gcontext.ClassifyDiff(lineCount, cfg.MaxDiffLines)

	// 超大型 diff 且未传 --force 时，阻塞等待用户确认。
	if level == gcontext.ExtraLarge && !flags.Force {
		fmt.Printf(
			"本次暂存变更约 %d 行，已超出建议范围。\n"+
				"Gita 将仅基于文件列表生成粗略的 commit message，建议：\n"+
				"1. 使用 git add -p 分批暂存后再执行 gita commit\n"+
				"2. 或使用 --force 强制生成（质量可能下降）\n"+
				"是否继续？[y/N] ",
			lineCount,
		)

		// 根据是否注入测试输入，选择读取源。
		stdin := io.Reader(os.Stdin)
		if testStdin != nil {
			stdin = testStdin
		}
		reader := bufio.NewReader(stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("读取用户输入失败: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			return fmt.Errorf("已取消：diff 过大")
		}
	}

	content := gcontext.BuildContext(diff, stat, fileNames, level)

	// 6. 加载并渲染 Prompt 模板。
	tmpl, err := prompt.LoadTemplate("commit")
	if err != nil {
		return fmt.Errorf("加载模板失败: %w", err)
	}

	// 命令行参数覆盖配置文件默认值。
	language := cfg.Language
	if flags.Lang != "" {
		language = flags.Lang
	}
	style := cfg.CommitStyle
	if flags.Style != "" {
		style = flags.Style
	}
	providerName := cfg.DefaultProvider
	if flags.Provider != "" {
		providerName = flags.Provider
	}

	// 将语言代码转为自然语言描述，让 LLM 更准确地遵循语言要求。
	// zh-CN → "中文"，en → "English"，其余原样传递。
	languageDisplay := language
	switch language {
	case "zh-CN":
		languageDisplay = "中文"
	case "en":
		languageDisplay = "English"
	}

	renderedPrompt := prompt.Render(tmpl, map[string]string{
		"diff":      content,
		"stat":      stat,
		"file_list": strings.Join(fileNames, "\n"),
		"hint":      flags.Hint,
		"language":  languageDisplay,
		"style":     style,
	})

	// 7. 创建 LLM Provider 并调用。
	// --api-key 临时覆盖环境变量，仅存于运行时内存，不落盘。
	// testProvider 非 nil 时直接使用（测试注入点），跳过真实 Provider 创建。
	var provider llm.Provider
	if testProvider != nil {
		provider = testProvider
	} else {
		var err error
		provider, err = llm.NewProvider(cfg, providerName, flags.APIKey)
		if err != nil {
			return fmt.Errorf("创建 LLM Provider 失败: %w", err)
		}
	}

	commitMsg, err := provider.Generate(context.Background(), renderedPrompt)
	if err != nil {
		return fmt.Errorf("生成 Commit Message 失败: %w", err)
	}

	// 8. 交互确认循环：Y 提交 / e 编辑 / r 重新生成 / n 取消。
	// 重新生成不限次数，每次视为全新请求（不复用旧连接的失败状态）。
	confirmer := newInteractConfirmer()
	currentMsg := commitMsg

	for {
		action, editedMsg, err := confirmer.confirm(currentMsg)
		if err != nil {
			return fmt.Errorf("交互确认失败: %w", err)
		}

		switch action {
		case interactConfirm:
			if err := execGitCommit(currentMsg); err != nil {
				return fmt.Errorf("git commit 执行失败: %w", err)
			}
			fmt.Println("✓ 提交成功")
			return nil

		case interactEdit:
			// 编辑后回到确认循环，让用户再次确认后再提交。
			currentMsg = editedMsg

		case interactRegenerate:
			// 重新生成：每次调用 LLM 视为全新请求，
			// 不限制次数，成本由用户承担。
			newMsg, err := provider.Generate(context.Background(), renderedPrompt)
			if err != nil {
				return fmt.Errorf("重新生成失败: %w", err)
			}
			currentMsg = newMsg

		case interactCancel:
			fmt.Println("已取消，未执行 git commit")
			return nil
		}
	}
}

// printHelp 输出 gita 的帮助信息。
func printHelp() {
	fmt.Println(`Gita —— AI 驱动的 Git 提交助手

用法:
  gita commit [flags]    基于 staged diff 生成 Commit Message（MVP）
  gita help              显示此帮助信息

MVP 阶段仅支持 commit 子命令，更多功能将在后续版本推出。
运行 'gita commit --help' 查看 commit 子命令的参数说明。`)
}
