package claude_adaptor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/render"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/controller"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"io"
	"net/http"
	"strings"
)

type Anthropic struct{}

func (a *Anthropic) DoRequest(c *gin.Context, request *anthropic.Request, meta *meta.Meta) (*anthropic.Usage, *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	reqBody, _ := common.GetRequestBody(c)
	if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
		logger.DebugForcef(ctx, "Antropic request: %s", string(reqBody))
	}
	requestBody := bytes.NewBuffer(reqBody)
	fullRequestURL := fmt.Sprintf("%s/v1/messages", meta.BaseURL)
	req, err := http.NewRequest(c.Request.Method, fullRequestURL, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return nil, openai.ErrorWrapper(fmt.Errorf("new request failed: %w", err), "do_request_failed", http.StatusInternalServerError)
	}
	err = setupRequestHeader(c, req, meta)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return nil, openai.ErrorWrapper(fmt.Errorf("setup request header failed: %w", err), "do_request_failed", http.StatusInternalServerError)
	}
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

func StreamHandler(c *gin.Context, resp *http.Response) (*anthropic.Usage, *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), "\n"); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	common.SetEventStreamHeaders(c)
	var inputTokens int
	var outputTokens int
	var cacheCreateTokens int
	var cacheHitTokens int
	for scanner.Scan() {
		data := scanner.Text()
		render.RawData(c, data)
		if len(data) < 6 || !strings.HasPrefix(data, "data:") {
			continue
		}
		data = strings.TrimPrefix(data, "data:")
		data = strings.TrimSpace(data)

		var claudeResponse anthropic.StreamResponse
		err := json.Unmarshal([]byte(data), &claudeResponse)
		if err != nil {
			logger.Error(ctx, "error unmarshalling stream response: "+err.Error())
			continue
		}
		if claudeResponse.Message != nil {
			inputTokens += claudeResponse.Message.Usage.InputTokens
			outputTokens += claudeResponse.Message.Usage.OutputTokens
			cacheCreateTokens += claudeResponse.Message.Usage.CacheCreationInputTokens
			cacheHitTokens += claudeResponse.Message.Usage.CacheReadInputTokens
		}
		if claudeResponse.Usage != nil {
			inputTokens += claudeResponse.Usage.InputTokens
			outputTokens += claudeResponse.Usage.OutputTokens
			cacheCreateTokens += claudeResponse.Usage.CacheCreationInputTokens
			cacheHitTokens += claudeResponse.Usage.CacheReadInputTokens
		}
	}
	usage := &anthropic.Usage{
		InputTokens:              inputTokens,
		OutputTokens:             outputTokens,
		CacheCreationInputTokens: cacheCreateTokens,
		CacheReadInputTokens:     cacheHitTokens,
	}
	if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
		logger.DebugForcef(c.Request.Context(), "claude usage: %v", usage)
	}
	if err := scanner.Err(); err != nil {
		logger.Error(ctx, "error reading stream: "+err.Error())
		return usage, openai.ErrorWrapper(err, "read_stream_failed", http.StatusInternalServerError)
	}
	err := resp.Body.Close()
	if err != nil {
		return usage, openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError)
	}
	return usage, nil
}

func Handler(c *gin.Context, resp *http.Response) (*anthropic.Usage, *model.ErrorWithStatusCode) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError)
	}
	var claudeResponse anthropic.Response
	err = json.Unmarshal(responseBody, &claudeResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if claudeResponse.Error != nil && claudeResponse.Error.Type != "" {
		return nil, &model.ErrorWithStatusCode{
			Error: model.Error{
				Message: claudeResponse.Error.Message,
				Type:    claudeResponse.Error.Type,
				Param:   "",
				Code:    claudeResponse.Error.Type,
			},
			StatusCode: resp.StatusCode,
		}
	}
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusRequestTimeout)
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError)
	}

	if config.DebugUserIds[c.GetInt(ctxkey.Id)] {
		logger.DebugForcef(c.Request.Context(), "claude usage: %v", claudeResponse.Usage)
	}
	return claudeResponse.Usage, nil
}

func setupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	req.Header.Set("x-api-key", meta.APIKey)
	anthropicVersion := c.Request.Header.Get("anthropic-version")
	if anthropicVersion == "" {
		anthropicVersion = "2023-06-01"
	}
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("anthropic-beta", "messages-2023-12-15")

	// https://x.com/alexalbert__/status/1812921642143900036
	// claude-3-5-sonnet can support 8k context
	if strings.HasPrefix(meta.ActualModelName, "claude-3-5-sonnet") {
		req.Header.Set("anthropic-beta", "max-tokens-3-5-sonnet-2024-07-15")
	}

	return nil
}

func isErrorHappened(meta *meta.Meta, resp *http.Response) bool {
	if resp == nil {
		if meta.ChannelType == channeltype.AwsClaude {
			return false
		}
		return true
	}
	if resp.StatusCode != http.StatusOK {
		return true
	}
	if meta.IsStream && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return true
	}
	return false
}
