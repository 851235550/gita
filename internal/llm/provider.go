// Package llm 定义与具体 LLM Provider 无关的统一调用接口。
// 所有 Provider（DeepSeek、OpenAI、Claude 等）均实现 Provider 接口，
// 上层业务代码通过接口调用，无需关心底层 API 差异（需求文档 4.2 节）。
package llm

import "context"

// Provider 定义 LLM 调用的统一接口。
// 各 Provider 实现负责：构造请求、发送 HTTP 调用、解析响应、处理错误。
// ctx 用于超时控制和请求取消，prompt 为已渲染完成的最终提示文本。
// 返回的 string 为 LLM 生成的文本内容（已去除 markdown 代码块等包装字符）。
type Provider interface {
	// Generate 向 LLM 发送 prompt 并返回生成的文本。
	// ctx 必须被各实现遵守，用于超时控制（需求文档建议阈值 30s）。
	Generate(ctx context.Context, prompt string) (string, error)
}
