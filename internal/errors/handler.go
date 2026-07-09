// Package errors 提供统一的异常处理与用户提示封装。
// 将需求文档第 8 节列出的 7 种异常场景映射为清晰的用户提示文案，
// 避免各模块各自构造提示信息导致文案不一致。
package errors

import (
	"errors"
	"fmt"
)

// 预定义的错误类型，便于上层用 errors.Is 判断具体异常场景。
var (
	// ErrNotGitRepo 当前目录不是有效的 Git 仓库。
	ErrNotGitRepo = errors.New("当前目录不是有效的 Git 仓库")

	// ErrNoStagedChanges 暂存区无变更。
	ErrNoStagedChanges = errors.New("没有检测到暂存变更，请先执行 git add")

	// ErrAPIKeyMissing API Key 未配置。
	ErrAPIKeyMissing = errors.New("API Key 未配置")

	// ErrLLMTimeout LLM 请求超时。
	ErrLLMTimeout = errors.New("LLM 请求超时")

	// ErrLLMEmptyContent LLM 返回空内容。
	ErrLLMEmptyContent = errors.New("LLM 返回内容为空")

	// ErrNetworkUnreachable 网络不可达。
	ErrNetworkUnreachable = errors.New("网络不可达")
)

// UserMessage 为各类异常返回面向用户的提示文案。
// 文案中需包含关键动作提示（如建议执行的命令、环境变量名等）。
func UserMessage(err error) string {
	switch {
	case errors.Is(err, ErrNotGitRepo):
		return "当前目录不是有效的 Git 仓库，请在 Git 仓库根目录或子目录下执行 gita commit"

	case errors.Is(err, ErrNoStagedChanges):
		return "没有检测到暂存变更，请先执行 git add 暂存文件后再执行 gita commit"

	case errors.Is(err, ErrAPIKeyMissing):
		return fmt.Sprintf("API Key 未配置: %v\n请设置对应的环境变量或使用 --api-key 参数", err)

	case errors.Is(err, ErrLLMTimeout):
		return fmt.Sprintf("LLM 请求超时（超过 30 秒）: %v\n请检查网络连接或稍后重试", err)

	case errors.Is(err, ErrLLMEmptyContent):
		return fmt.Sprintf("LLM 返回内容为空: %v\n可尝试重新生成（在确认界面输入 r）或手动执行 git commit", err)

	case errors.Is(err, ErrNetworkUnreachable):
		return fmt.Sprintf("网络不可达: %v\n请检查网络连接，或使用 git commit 命令手动提交", err)

	default:
		return fmt.Sprintf("发生未知错误: %v", err)
	}
}

// Wrap 将原始错误与预定义错误类型关联，保留原始错误链。
// 使用 fmt.Errorf + %w 确保 errors.Is 能正确匹配。
func Wrap(sentinel error, detail string) error {
	return fmt.Errorf("%s: %w", detail, sentinel)
}

// Wrapf 同 Wrap，但支持格式化字符串。
func Wrapf(sentinel error, format string, args ...interface{}) error {
	return Wrap(sentinel, fmt.Sprintf(format, args...))
}
