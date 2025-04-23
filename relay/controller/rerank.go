package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"io"
	"net/http"
)

func RelayRerankHelper(c *gin.Context, mode int) *relaymodel.ErrorWithStatusCode {
	ctx := c.Request.Context()
	meta := meta.GetByContext(c)
	rerankRequest, err := getRerankRequest(c, mode)
	if err != nil {
		logger.Errorf(ctx, "getRerankRequest failed: %s", err.Error())
		return openai.ErrorWrapper(err, "invalid_rerank_request", http.StatusBadRequest)
	}

	// map model name
	meta.OriginModelName = rerankRequest.Model
	rerankRequest.Model, _ = getMappedModelName(rerankRequest.Model, meta.ModelMapping)
	meta.ActualModelName = rerankRequest.Model

	// model validation
	bizErr := validateRerankRequest(rerankRequest, meta)
	if bizErr != nil {
		return bizErr
	}
	// valid account
	userQuota, err := model.CacheGetUserQuota(ctx, meta.UserId)
	if err != nil {
		return openai.ErrorWrapper(err, "get_user_quota_failed", http.StatusInternalServerError)
	}
	if userQuota <= 0 {
		return openai.ErrorWrapper(errors.New("user quota is not enough"), "insufficient_user_quota", http.StatusForbidden)
	}

	rerankAdaptor := relay.GetRerankAdaptor(meta.ChannelType)
	if rerankAdaptor == nil {
		return openai.ErrorWrapper(errors.New("rerankAdaptor is nil"), "adaptor_nil", http.StatusInternalServerError)
	}
	rerankAdaptor.Init(meta)

	// get request body
	requestBody, err := getRerankRequestBody(c, meta, rerankRequest, rerankAdaptor)
	if err != nil {
		return openai.ErrorWrapper(err, "convert_request_failed", http.StatusInternalServerError)
	}

	// do request
	resp, err := rerankAdaptor.DoRequest(c, meta, requestBody)
	if err != nil {
		logger.Errorf(ctx, "DoRequest failed: %s", err.Error())
		return openai.ChannelErrorWrapper(err, "do_request_failed", http.StatusInternalServerError)
	}
	if isErrorHappened(meta, resp) {
		return RelayErrorHandler(resp)
	}
	// do response
	usage, respErr := rerankAdaptor.DoRerankResponse(
		c, resp, meta)
	if respErr != nil {
		logger.Errorf(ctx, "respErr is not nil: %+v", respErr)
		return respErr
	}

	// consume quota
	go postConsumeRerankQuota(c, ctx, usage, meta)
	return nil
}

func getRerankRequestBody(c *gin.Context, m *meta.Meta, request *relaymodel.RerankRequest, adaptor adaptor.RerankAdaptor) (io.Reader, error) {
	var requestBody io.Reader
	convertedRequest, err := adaptor.ConvertRerankRequest(request)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request failed: %s\n", err.Error())
		return nil, err
	}
	jsonData, err := json.Marshal(convertedRequest)
	if err != nil {
		logger.Debugf(c.Request.Context(), "converted request json_marshal_failed: %s\n", err.Error())
		return nil, err
	}
	logger.Debugf(c.Request.Context(), "converted request: \n%s", string(jsonData))
	requestBody = bytes.NewBuffer(jsonData)
	return requestBody, nil
}

func validateRerankRequest(rerankRequest *relaymodel.RerankRequest, meta *meta.Meta) *relaymodel.ErrorWithStatusCode {
	// check prompt length
	if rerankRequest.Query == "" {
		return openai.ErrorWrapper(errors.New("query is required"), "query_missing", http.StatusBadRequest)
	}
	if len(rerankRequest.Documents) == 0 {
		return openai.ErrorWrapper(errors.New("documents is empty"), "documents_empty", http.StatusBadRequest)
	}
	return nil
}

func getRerankRequest(c *gin.Context, mode int) (*relaymodel.RerankRequest, error) {
	rerankRequest := &relaymodel.RerankRequest{}
	err := common.UnmarshalBodyReusable(c, rerankRequest)
	if err != nil {
		return nil, err
	}
	return rerankRequest, nil
}
