package claude_adaptor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/adaptor/vertexai"
	claude "github.com/songquanpeng/one-api/relay/adaptor/vertexai/claude"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"net/http"
)

const anthropicVersion = "vertex-2023-10-16"

var _ Adaptor = new(Vertextai)

type Vertextai struct{}

func (a *Vertextai) DoRequest(c *gin.Context, request *anthropic.Request, meta *meta.Meta) (*anthropic.Usage, *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	fullRequestURL, err := GetRequestURL(meta)
	claudeReq := claude.Request{
		AnthropicVersion: anthropicVersion,
		// Model:            claudeReq.Model,
		Messages:    request.Messages,
		System:      request.System,
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		TopP:        request.TopP,
		TopK:        request.TopK,
		Stream:      request.Stream,
		Tools:       request.Tools,
	}
	jsonData, _ := json.Marshal(claudeReq)
	requestBody := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest(c.Request.Method, fullRequestURL, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return nil, openai.ErrorWrapper(fmt.Errorf("new request failed: %w", err), "do_request_failed", http.StatusInternalServerError)
	}
	adaptor.SetupCommonRequestHeader(c, req, meta)
	token, err := vertexai.GetToken(c, meta.ChannelId, meta.Config.VertexAIADC)
	if err != nil {
		logger.Errorf(ctx, "GetToken failed: %s", err.Error())
		return nil, openai.ErrorWrapper(fmt.Errorf("get token failed: %w", err), "get_token_failed", http.StatusInternalServerError)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := adaptor.DoRequest(c, req)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return nil, openai.ErrorWrapper(fmt.Errorf("do request failed: %s", adaptor.MaskBaseURL(err.Error(), meta.BaseURL)), "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(meta, resp) {
		return nil, controller.RelayErrorHandler(resp)
	}

	if meta.IsStream {
		return StreamHandler(c, resp)
	} else {
		return Handler(c, resp)
	}
}

func GetRequestURL(meta *meta.Meta) (string, error) {
	suffix := ""
	if meta.IsStream {
		suffix = "streamRawPredict?alt=sse"
	} else {
		suffix = "rawPredict"
	}

	if meta.BaseURL != "" {
		return fmt.Sprintf(
			"%s/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
			meta.BaseURL,
			meta.Config.VertexAIProjectID,
			meta.Config.Region,
			meta.ActualModelName,
			suffix,
		), nil
	}
	return fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
		meta.Config.Region,
		meta.Config.VertexAIProjectID,
		meta.Config.Region,
		meta.ActualModelName,
		suffix,
	), nil
}
