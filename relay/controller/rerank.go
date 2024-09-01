package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
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

	//todo
	return nil
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
