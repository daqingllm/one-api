package gemini

import (
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
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
