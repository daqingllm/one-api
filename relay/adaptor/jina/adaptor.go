package jina

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/relaymode"
	"io"
	"net/http"
)

type Adaptor struct {
}

func (a Adaptor) ConvertRerankRequest(request *model.RerankRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return &JinaRerankRequest{
		Documents:       request.Documents,
		Query:           request.Query,
		Model:           request.Model,
		TopN:            request.TopN,
		ReturnDocuments: false,
	}, nil
}

func (a Adaptor) DoRerankResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.RerankUsage, err *model.ErrorWithStatusCode) {
	err, usage = RerankHandler(c, resp)
	return
}

func RerankHandler(c *gin.Context, resp *http.Response) (*model.ErrorWithStatusCode, *model.RerankUsage) {
	var rerankResp model.RerankResponse
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}
	err = json.Unmarshal(responseBody, &rerankResp)
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError), nil
	}

	jsonResponse, err := json.Marshal(rerankResp)
	// Reset response body
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(resp.StatusCode)
	//_, err = io.Copy(c.Writer, resp.Body)
	_, err = c.Writer.Write(jsonResponse)
	if err != nil {
		return openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusRequestTimeout), nil
	}
	err = resp.Body.Close()
	if err != nil {
		return openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError), nil
	}

	return nil, &rerankResp.Usage
}

func (a Adaptor) Init(_ *meta.Meta) {
}

func (a Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	if meta.Mode == relaymode.ChatCompletions {
		return "https://deepsearch.jina.ai/v1/chat/completions", nil
	}
	if meta.Mode == relaymode.Embeddings {
		return "https://api.jina.ai/v1/embeddings", nil
	}
	if meta.Mode == relaymode.Rerank {
		return "https://api.jina.ai/v1/rerank", nil
	}
	return "", fmt.Errorf("unsupported relay mode %d for Jina", meta.Mode)
}

func (a Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) error {
	adaptor.SetupCommonRequestHeader(c, req, meta)
	req.Header.Set("Authorization", "Bearer "+meta.APIKey)
	return nil
}

func (a Adaptor) ConvertRequest(c *gin.Context, meta *meta.Meta, request *model.GeneralOpenAIRequest) (any, error) {

	if request == nil {
		return nil, errors.New("request is nil")
	}
	switch meta.Mode {
	case relaymode.Embeddings:
		jinaEmbeddingRequest := ConvertEmbeddingRequest(*request)
		return jinaEmbeddingRequest, nil
	case relaymode.ChatCompletions:
		jinaChatRequest := ConvertChatRequest(*request)
		return jinaChatRequest, nil
	default:
		return nil, fmt.Errorf("unsupported relay mode %d for Jina", meta.Mode)
	}
}

func (a Adaptor) ConvertImageRequest(request *model.ImageRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return request, nil
}

func (a Adaptor) DoRequest(c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	return adaptor.DoRequestHelper(a, c, meta, requestBody)
}

func (a Adaptor) DoResponse(c *gin.Context, resp *http.Response, meta *meta.Meta) (usage *model.Usage, err *model.ErrorWithStatusCode) {
	if meta.IsStream {
		var responseText string
		err, responseText, usage = openai.StreamHandler(c, resp, meta.Mode)
		if usage == nil || usage.TotalTokens == 0 {
			usage = openai.ResponseText2Usage(responseText, meta.ActualModelName, meta.PromptTokens)
		}
		if usage.TotalTokens != 0 && usage.PromptTokens == 0 { // some channels don't return prompt tokens & completion tokens
			logger.Error(context.Background(), fmt.Sprintf("Usage tokens maybe abnormal, response=%s, meta=%+v", responseText, *meta))
			usage.PromptTokens = meta.PromptTokens
			usage.CompletionTokens = usage.TotalTokens - meta.PromptTokens
		}
	} else {
		switch meta.Mode {
		case relaymode.ChatCompletions:
			err, usage = openai.ChatHandler(c, resp, meta.PromptTokens, meta.ActualModelName)
		default:
			err, usage = openai.HandlerWithRawResp(c, resp, meta.PromptTokens, meta.ActualModelName)
		}
	}
	return
}

func (a Adaptor) GetModelList() []string {
	return ModelList
}

func (a Adaptor) GetChannelName() string {
	return "jina"
}
