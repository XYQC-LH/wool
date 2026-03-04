package scheduler

import (
	"context"

	"nexus-api/internal/model"
)

// RequestExecutor 请求执行器函数类型（非流式）
type RequestExecutor func(ctx context.Context, provider *model.ModelProvider) (interface{}, error)

// StreamExecutor 流式请求执行器（支持首包回调）
// ⭐ 修复：添加 onFirstChunk 回调参数，用于通知首包到达
type StreamExecutor func(ctx context.Context, provider *model.ModelProvider, onFirstChunk func()) error

// ChatCompletionRequest 聊天完成请求
// ⭐ 核心类型：定义聊天完成请求的结构
type ChatCompletionRequest struct {
	// 模型ID
	Model string `json:"model"`

	// 消息列表
	Messages []map[string]interface{} `json:"messages"`

	// 最大生成token数
	MaxTokens int `json:"max_tokens,omitempty"`

	// 温度参数（0-2）
	Temperature float64 `json:"temperature,omitempty"`

	// 是否流式输出
	Stream bool `json:"stream,omitempty"`

	// Top P采样参数
	TopP float64 `json:"top_p,omitempty"`

	// 频率惩罚
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`

	// 存在惩罚
	PresencePenalty float64 `json:"presence_penalty,omitempty"`

	// 停止序列
	Stop []string `json:"stop,omitempty"`

	// 用户标识
	User string `json:"user,omitempty"`
}

// ChatCompletionMessage 聊天消息
type ChatCompletionMessage struct {
	// 角色：system, user, assistant
	Role string `json:"role"`

	// 消息内容
	Content string `json:"content"`

	// 消息名称（可选）
	Name string `json:"name,omitempty"`
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	// 响应ID
	ID string `json:"id"`

	// 对象类型
	Object string `json:"object"`

	// 创建时间戳
	Created int64 `json:"created"`

	// 模型ID
	Model string `json:"model"`

	// 选择列表
	Choices []ChatCompletionChoice `json:"choices"`

	// 使用情况
	Usage ChatCompletionUsage `json:"usage"`

	// 系统指纹
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// ChatCompletionChoice 聊天完成选择
type ChatCompletionChoice struct {
	// 索引
	Index int `json:"index"`

	// 消息
	Message ChatCompletionMessage `json:"message,omitempty"`

	// Delta（流式响应）
	Delta map[string]interface{} `json:"delta,omitempty"`

	// 完成原因
	FinishReason string `json:"finish_reason"`
}

// ChatCompletionUsage 聊天完成使用情况
type ChatCompletionUsage struct {
	// Prompt tokens
	PromptTokens int `json:"prompt_tokens"`

	// Completion tokens
	CompletionTokens int `json:"completion_tokens"`

	// 总tokens
	TotalTokens int `json:"total_tokens"`
}

// StreamChunk 流式响应块
type StreamChunk struct {
	// 响应ID
	ID string `json:"id"`

	// 对象类型
	Object string `json:"object"`

	// 创建时间戳
	Created int64 `json:"created"`

	// 模型ID
	Model string `json:"model"`

	// 选择列表
	Choices []StreamChoice `json:"choices"`

	// 系统指纹
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
}

// StreamChoice 流式选择
type StreamChoice struct {
	// 索引
	Index int `json:"index"`

	// Delta
	Delta map[string]interface{} `json:"delta"`

	// 完成原因
	FinishReason string `json:"finish_reason,omitempty"`
}
