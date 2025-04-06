package aws

import "github.com/songquanpeng/one-api/relay/adaptor/anthropic"

// Request is the request to AWS Claude
//
// https://docs.aws.amazon.com/bedrock/latest/userguide/model-parameters-anthropic-claude-messages.html
type Request struct {
	// AnthropicVersion should be "bedrock-2023-05-31"
	AnthropicVersion string              `json:"anthropic_version"`
	Messages         []anthropic.Message `json:"messages"`
	System           any                 `json:"system,omitempty"`
	MaxTokens        int                 `json:"max_tokens,omitempty"`
	Temperature      *float64            `json:"temperature,omitempty"`
	Thinking         *anthropic.Thinking `json:"thinking,omitempty"`
	TopP             *float64            `json:"top_p,omitempty"`
	TopK             int                 `json:"top_k,omitempty"`
	StopSequences    []string            `json:"stop_sequences,omitempty"`
	Tools            []anthropic.Tool    `json:"tools,omitempty"`
	ToolChoice       any                 `json:"tool_choice,omitempty"`
	AnthropicBeta    []string            `json:"anthropic_beta,omitempty"`
}
