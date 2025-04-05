package ideogram

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/tidwall/gjson"
)

type IdeoGramModelRetriever struct {
}

func (r *IdeoGramModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {

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
		modelName = gjson.GetBytes(bodyBytes, "image_request.model").String()
		if modelName == "" {
			return "", nil
		}
		return modelName, nil

	} else if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {

		err := srcCtx.Request.ParseMultipartForm(32 << 20)
		if err != nil {
			return "", &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusBadRequest,
				Error:      relaymodel.Error{Message: "invalid_form_request", Code: "INVALID_FORM"},
			}
		}
		modelName := srcCtx.Request.Form.Get("model")
		return modelName, nil
	}

	return "", &relaymodel.ErrorWithStatusCode{
		StatusCode: http.StatusUnsupportedMediaType,
		Error:      relaymodel.Error{Message: "unsupported_content_type", Code: "UNSUPPORTED_CONTENT_TYPE"},
	}
}
