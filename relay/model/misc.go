package model

type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type PromptTokensDetails struct {
	AudioTokens  *int `json:"audio_tokens,omitempty"`
	CachedTokens *int `json:"cached_tokens,omitempty"`
}

type CompletionTokensDetails struct {
	AcceptedPredictionTokens *int `json:"accepted_prediction_tokens,omitempty"`
	AudioTokens              *int `json:"audio_tokens,omitempty"`
	ReasoningTokens          *int `json:"reasoning_tokens,omitempty"`
	RejectedPredictionTokens *int `json:"rejected_prediction_tokens,omitempty"`
}

type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    any    `json:"code"`
}

type ErrorWithStatusCode struct {
	IsChannelResponseError bool `json:"is_channel_response_error"`
	Error
	StatusCode int `json:"status_code"`
}

func NewErrorWithStatusCode(statusCode int, code any, message string) *ErrorWithStatusCode {
	return &ErrorWithStatusCode{
		IsChannelResponseError: false,
		Error: Error{
			Message: message,
			Type:    "",
			Param:   "",
			Code:    code,
		},
		StatusCode: statusCode,
	}
}
