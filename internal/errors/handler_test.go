package errors

import (
	"errors"
	"strings"
	"testing"
)

// ============================================================================
// TC-T5.1: 7 种异常场景逐一验证
// ============================================================================

func TestUserMessage_NotGitRepo(t *testing.T) {
	// Arrange
	err := Wrap(ErrNotGitRepo, "rev-parse 失败")

	// Act
	msg := UserMessage(err)

	// Assert: 应包含"不是有效的 Git 仓库"关键字。
	if !strings.Contains(msg, "不是有效的 Git 仓库") {
		t.Errorf("文案缺少关键字 '不是有效的 Git 仓库': %s", msg)
	}
}

func TestUserMessage_NoStagedChanges(t *testing.T) {
	// Arrange
	err := Wrap(ErrNoStagedChanges, "暂存区为空")

	// Act
	msg := UserMessage(err)

	// Assert: 应包含"git add"关键字。
	if !strings.Contains(msg, "git add") {
		t.Errorf("文案缺少关键字 'git add': %s", msg)
	}
}

func TestUserMessage_APIKeyMissing(t *testing.T) {
	// Arrange
	err := Wrap(ErrAPIKeyMissing, "GITA_DEEPSEEK_API_KEY 环境变量未设置")

	// Act
	msg := UserMessage(err)

	// Assert: 应包含环境变量名。
	if !strings.Contains(msg, "GITA_DEEPSEEK_API_KEY") {
		t.Errorf("文案应包含环境变量名: %s", msg)
	}
	if !strings.Contains(msg, "--api-key") {
		t.Errorf("文案应提示 --api-key 参数: %s", msg)
	}
}

func TestUserMessage_LLMTimeout(t *testing.T) {
	// Arrange
	err := Wrap(ErrLLMTimeout, "请求超过 30 秒")

	// Act
	msg := UserMessage(err)

	// Assert: 应包含"超时"和重试提示。
	if !strings.Contains(msg, "超时") {
		t.Errorf("文案缺少关键字 '超时': %s", msg)
	}
	if !strings.Contains(msg, "重试") {
		t.Errorf("文案应包含重试提示: %s", msg)
	}
}

func TestUserMessage_LLMEmptyContent(t *testing.T) {
	// Arrange
	err := Wrap(ErrLLMEmptyContent, "响应 choices 为空")

	// Act
	msg := UserMessage(err)

	// Assert: 应包含"内容为空"和重新生成提示。
	if !strings.Contains(msg, "内容为空") {
		t.Errorf("文案缺少关键字 '内容为空': %s", msg)
	}
	if !strings.Contains(msg, "重新生成") {
		t.Errorf("文案应包含重新生成提示: %s", msg)
	}
}

func TestUserMessage_NetworkUnreachable(t *testing.T) {
	// Arrange
	err := Wrap(ErrNetworkUnreachable, "无法连接到 api.deepseek.com")

	// Act
	msg := UserMessage(err)

	// Assert: 应包含"网络"关键字和手动提交提示。
	if !strings.Contains(msg, "网络") {
		t.Errorf("文案缺少关键字 '网络': %s", msg)
	}
	if !strings.Contains(msg, "git commit") {
		t.Errorf("文案应包含手动提交提示: %s", msg)
	}
}

func TestUserMessage_UnknownError(t *testing.T) {
	// Arrange
	err := errors.New("some unexpected error")

	// Act
	msg := UserMessage(err)

	// Assert: 未知错误应有通用提示。
	if !strings.Contains(msg, "未知错误") {
		t.Errorf("未知错误应包含'未知错误'提示: %s", msg)
	}
}

func TestWrap_PreservesSentinel(t *testing.T) {
	// Arrange
	err := Wrap(ErrNotGitRepo, "some detail")

	// Act & Assert: errors.Is 应能识别原始 sentinel。
	if !errors.Is(err, ErrNotGitRepo) {
		t.Error("Wrap 后的错误应能被 errors.Is 匹配")
	}
}

func TestWrapf_PreservesSentinel(t *testing.T) {
	// Arrange
	err := Wrapf(ErrAPIKeyMissing, "变量 %s 为空", "GITA_OPENAI_API_KEY")

	// Act & Assert
	if !errors.Is(err, ErrAPIKeyMissing) {
		t.Error("Wrapf 后的错误应能被 errors.Is 匹配")
	}
	if !strings.Contains(err.Error(), "GITA_OPENAI_API_KEY") {
		t.Errorf("错误信息应包含格式化后的详情: %v", err)
	}
}
