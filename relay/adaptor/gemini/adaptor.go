package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/songquanpeng/one-api/common/logger"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	channelhelper "github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
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
	} else if strings.HasPrefix(meta.ActualModelName, "gemini-2.0-flash-thinking-exp") {
		defaultVersion = "v1alpha"
	}

	version := helper.AssignOrDefault(meta.Config.APIVersion, defaultVersion)
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
	if strings.HasPrefix(meta.ActualModelName, "gemini-2.0-flash-thinking-exp") {
		logger.DebugForcef(context.Background(), "gemini-2.0-flash-thinking-exp url: %s", url)
	}
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
		if meta.OriginModelName == "gemini-2.0-flash-exp-search" {
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
		if strings.HasPrefix(meta.ActualModelName, "gemini-2.0-flash-thinking-exp") {
			jsonData, _ := json.Marshal(geminiRequest)
			logger.DebugForcef(context.Background(), "gemini-2.0-flash-thinking-exp request: %s", jsonData)
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
		var responseText string
		err, responseText = StreamHandler(c, resp, meta.ActualModelName)
		usage = openai.ResponseText2Usage(responseText, meta.ActualModelName, meta.PromptTokens)
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
