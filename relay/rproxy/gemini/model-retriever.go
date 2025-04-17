package gemini

import (
	"net/http"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type GeminiModelRetriever struct {
}

func (r *GeminiModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	model := context.SrcContext.Param("model")
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
