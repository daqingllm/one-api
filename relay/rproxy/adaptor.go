package rproxy

import (
	"bufio"
	"encoding/json"
	"strings"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/render"
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type RproxyAdaptor interface {
	GetChannel() *model.Channel
	DoRequest(context *RproxyContext) (err *relaymodel.ErrorWithStatusCode)
}

func StreamHandler(context *RproxyContext) (err *relaymodel.ErrorWithStatusCode) {
	srcContext := context.SrcContext
	ctx := context.SrcContext.Request.Context()
	scanner := bufio.NewScanner(context.SrcContext.Request.Response.Body)
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
	common.SetEventStreamHeaders(context.SrcContext)
	var inputTokens int
	var outputTokens int
	var cacheCreateTokens int
	var cacheHitTokens int
	for scanner.Scan() {
		data := scanner.Text()
		render.RawData(srcContext, data)
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
	if config.DebugUserIds[srcContext.GetInt(ctxkey.Id)] {
		logger.DebugForcef(srcContext.Request.Context(), "claude usage: %v", usage)
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
