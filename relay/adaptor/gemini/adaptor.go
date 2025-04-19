package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	channelhelper "github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

type Adaptor struct {
}

func (a *Adaptor) Init(meta *meta.Meta) {

}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	defaultVersion := config.GeminiVersion
	if strings.HasPrefix(meta.ActualModelName, "gemini-2.0-flash-exp") {
		defaultVersion = "v1beta"
	}

	version := helper.AssignOrDefault(meta.Config.APIVersion, defaultVersion)
	if strings.HasPrefix(meta.ActualModelName, "gemini-2.0-flash-thinking-exp") {
		version = "v1alpha"
	}
	action := ""
	switch meta.Mode {
	case relaymode.Embeddings:
		action = "batchEmbedContents"
	default:
		action = "generateContent"
	}

	if meta.IsStream {
		action = "streamGenerateContent?alt=sse"
	}

	url := fmt.Sprintf("%s/%s/models/%s:%s", meta.BaseURL, version, meta.ActualModelName, action)
	return url, nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	channelhelper.SetupCommonRequestHeader(c, req, meta)
	req.Header.Set("x-goog-api-key", meta.APIKey)
	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, meta *meta.Meta, request *model.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	switch meta.Mode {
	case relaymode.Embeddings:
		geminiEmbeddingRequest := ConvertEmbeddingRequest(*request)
		return geminiEmbeddingRequest, nil
	default:
		geminiRequest := ConvertRequest(*request)
		//if meta.OriginModelName == "gemini-2.0-flash-exp-search" {
		if strings.HasSuffix(meta.OriginModelName, "-search") || request.WebSearchOptions != nil {
			meta.Extra["web_search"] = "true"
			if config.DebugUserIds[meta.UserId] {
				logger.DebugForcef(c.Request.Context(), "web search: %s", meta.ActualModelName)
			}
			if geminiRequest.Tools == nil {
				geminiRequest.Tools = []ChatTools{
					{
						GoogleSearch: &Empty{},
					},
				}
			} else {
				geminiRequest.Tools = append(geminiRequest.Tools, ChatTools{
					GoogleSearch: &Empty{},
				})
			}
		} else if strings.HasPrefix(meta.ActualModelName, "gemini-2.0-flash-thinking-exp") {
			geminiRequest.GenerationConfig.ThinkingConfig = &ThinkingConfig{IncludeThoughts: true}
		}
		if strings.HasSuffix(meta.OriginModelName, "-nothink") {
			geminiRequest.GenerationConfig.ThinkingConfig = &ThinkingConfig{IncludeThoughts: false, ThinkingBudget: 0}
		}
		if config.DebugUserIds[meta.UserId] {
			geminiRequestJSON, _ := json.Marshal(geminiRequest)
			logger.DebugForcef(c.Request.Context(), "gemini request: %s ,uid: %d", string(geminiRequestJSON), meta.UserId)
		}
		return geminiRequest, nil
	}
}

func (a *Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return channelhelper.DoRequestHelper(a, c, meta, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	if meta.IsStream {
		err, usage = StreamHandler(c, resp, meta.ActualModelName)
	} else {
		switch meta.Mode {
		case relaymode.Embeddings:
			err, usage = EmbeddingHandler(c, resp)
		default:
			err, usage = Handler(c, resp, meta.PromptTokens, meta.ActualModelName)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "google gemini"
}
