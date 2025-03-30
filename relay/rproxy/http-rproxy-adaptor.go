package rproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/model"
)

type HttpRproxyAdaptor struct {
	GetRequestUrl        func(context *RproxyContext) string
	HandlerRequestHeader func(context *RproxyContext) map[string]string
}

func (a HttpRproxyAdaptor) DoRequest(context *RproxyContext) (err *model.ErrorWithStatusCode) {
	jsonData, _ := json.Marshal(context.GetRequest())
	requestBody := bytes.NewBuffer(jsonData)
	if config.DebugUserIds[context.GetUserId()] {
		logger.DebugForcef(context.SrcContext, "request: %s", string(jsonData))
	}
	req, e := http.NewRequest(context.GetRequest().Method, a.GetRequestUrl(context), requestBody)
	if e != nil {
		logger.Errorf(context.SrcContext, "DoRequest failed: %s", e.Error())
		return nil, openai.ErrorWrapper(fmt.Errorf("new request failed: %w", err), "do_request_failed", http.StatusInternalServerError)
	}
	headerParams := a.HandlerRequestHeader(context)
	err = setupRequestHeader(c, req, meta)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return nil, openai.ErrorWrapper(fmt.Errorf("setup request header failed: %w", err), "do_request_failed", http.StatusInternalServerError)
	}
	for key, value := range headerParams {
		req.Header.Set(key, value)
	}
	resp, e := adaptor.DoRequest(context.SrcContext, req)
	if e != nil {
		logger.Errorf(context.SrcContext, "DoRequest failed: %s", e.Error())
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
	return nil
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
