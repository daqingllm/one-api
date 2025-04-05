package ideogram

import (
	"net/http"
	"strings"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/tidwall/gjson"
)

type IdeoGramEditModelRetriever struct {
}

func (r *IdeoGramEditModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	srcCtx := context.SrcContext
	contentType := srcCtx.Request.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		err := srcCtx.Request.ParseMultipartForm(32 << 20)
		if err != nil {
			return "", &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusBadRequest,
				Error:      relaymodel.Error{Message: "invalid_form_request", Code: "INVALID_FORM"},
			}
		}
		image_request := srcCtx.Request.Form.Get("image_request")
		modelName = gjson.Get(image_request, "model").String()
		return modelName, nil
	}

	return "", &relaymodel.ErrorWithStatusCode{
		StatusCode: http.StatusUnsupportedMediaType,
		Error:      relaymodel.Error{Message: "unsupported_content_type", Code: "UNSUPPORTED_CONTENT_TYPE"},
	}
}
