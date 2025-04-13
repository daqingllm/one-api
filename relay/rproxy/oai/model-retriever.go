package oai

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/tidwall/gjson"
)

type OAIModelRetriever struct {
}

func (r *OAIModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {

	srcCtx := context.SrcContext
	contentType := srcCtx.Request.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		bodyBytes, err := io.ReadAll(srcCtx.Request.Body)
		if err != nil {
			return "", &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusBadRequest,
				Error:      relaymodel.Error{Message: "read_request_body_failed", Code: "READ_BODY_FAILED"},
			}
		}
		srcCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		modelName = gjson.GetBytes(bodyBytes, "model").String()
		if modelName == "" {
			return "", nil
		}
		//todo  fixme
		context.ResolvedRequest = bodyBytes
		return modelName, nil
	}
	return "", &relaymodel.ErrorWithStatusCode{
		StatusCode: http.StatusUnsupportedMediaType,
		Error:      relaymodel.Error{Message: "unsupported_content_type", Code: "UNSUPPORTED_CONTENT_TYPE"},
	}
}
