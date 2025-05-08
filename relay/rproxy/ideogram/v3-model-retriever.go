package ideogram

import (
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type IdeoGramV3ModelRetriever struct {
}

func (r *IdeoGramV3ModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	model := context.SrcContext.Param("model")
	if model == "" {
		return "", &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusBadRequest,
			Error: relaymodel.Error{
				Message: "invalid_path",
				Code:    "INVALID_PATH",
			},
		}
	}
	parts := strings.Split(model, "-")
	if len(parts) != 2 {
		return "", &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusBadRequest,
			Error: relaymodel.Error{
				Message: "invalid_path",
				Code:    "INVALID_PATH",
			},
		}
	}
	modelName = strings.ToUpper(parts[1])
	return modelName, nil
}
