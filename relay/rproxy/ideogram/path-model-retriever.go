package ideogram

import (
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type IdeoGramPathModelRetriever struct {
}

func (r *IdeoGramPathModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	path := context.SrcContext.Request.URL.Path

	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	if len(parts) == 0 {
		return "", &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusBadRequest,
			Error: relaymodel.Error{
				Message: "invalid_path",
				Code:    "INVALID_PATH",
			},
		}
	}

	lastSegment := parts[len(parts)-1]
	modelName = strings.ToUpper(lastSegment)

	return modelName, nil
}
