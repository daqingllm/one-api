package gemini

import (
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

type GeminiModelRetriever struct {
}

func (r *GeminiModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	modelAction := context.SrcContext.Param("modelAction")
	parts := strings.SplitN(modelAction, ":", 2)
	model := parts[0]
	if model == "" {
		return "", &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusBadRequest,
			Error: relaymodel.Error{
				Message: "model parameter is required",
				Code:    "MISSING_MODEL_PARAM",
			},
		}
	}
	return model, nil
}

type GeminiCacheModelRetriever struct {
}

func (r *GeminiCacheModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	retriever := &common.DefaultModelRetriever{}
	modelName, err = retriever.Retrieve(context)
	if err != nil {
		return "", err
	}
	if strings.Contains(modelName, "/") {
		parts := strings.Split(modelName, "/")
		if len(parts) > 1 {
			modelName = parts[1]
		}
	}
	return modelName, nil

}
