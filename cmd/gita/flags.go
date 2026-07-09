package main

import (
	"flag"
	"fmt"
	"os"
)

// commitFlags 保存 gita commit 命令的所有 CLI 参数。
// 各参数均可覆盖 config.yaml 中的默认值（需求文档 6.2 节）。
type commitFlags struct {
	// Hint 额外上下文说明，拼入 prompt 提升生成质量。
	Hint string

	// Lang 覆盖默认输出语言（zh-CN / en）。
	Lang string

	// Style 覆盖默认输出风格（conventional / plain）。
	Style string

	// Provider 覆盖默认 LLM Provider 名称。
	Provider string

	// APIKey 临时指定的 Key，仅本次生效，不落盘。
	APIKey string

	// Force 超大 diff 时跳过确认提示，强制生成。
	Force bool
}

// parseCommitFlags 解析 gita commit 的命令行参数。
// args 应为 os.Args 中"commit"之后的部分。
func parseCommitFlags(args []string) (*commitFlags, error) {
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)

	// 禁止 flag 包自动打印错误到 stderr，
	// 我们自行控制错误格式和退出码。
	fs.SetOutput(&nullWriter{})

	flags := &commitFlags{}

	fs.StringVar(&flags.Hint, "hint", "", "额外上下文说明（如 'hotfix'），提升生成质量")
	fs.StringVar(&flags.Lang, "lang", "", "覆盖默认输出语言（zh-CN / en）")
	fs.StringVar(&flags.Style, "style", "", "覆盖默认输出风格（conventional / plain）")
	fs.StringVar(&flags.Provider, "provider", "", "覆盖默认 LLM Provider（deepseek / openai / claude）")
	fs.StringVar(&flags.APIKey, "api-key", "", "临时指定 API Key，仅本次生效，不写入文件")
	fs.BoolVar(&flags.Force, "force", false, "超大 diff 时跳过确认提示，强制生成")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return flags, nil
}

// printCommitHelp 输出 gita commit 子命令的帮助信息。
func printCommitHelp() {
	fmt.Println(`gita commit —— 基于 staged diff 生成 Commit Message

用法:
  gita commit [flags]

Flags:
  --hint <text>        额外上下文说明（如传 "hotfix"），拼入 prompt
  --lang <zh-CN|en>    覆盖 config.yaml 中的默认输出语言
  --style <style>      覆盖默认 Commit Message 风格（conventional | plain）
  --provider <name>    覆盖默认 LLM Provider（deepseek | openai | claude）
  --api-key <key>      临时指定 API Key，仅本次生效，不写入任何文件
  --force              超大 diff 时跳过确认提示，强制生成

示例:
  git add .
  gita commit --hint "为兼容旧版 API 保留了废弃字段"`)
}

// nullWriter 是一个丢弃所有写入的 io.Writer，用于抑制 flag 包的默认错误输出。
type nullWriter struct{}

func (w *nullWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

// 确保 os.Stdout / os.Stderr 可用（测试中可能被重定向）。
var (
	stdout = os.Stdout
	stderr = os.Stderr
)
