package common

import (
	"net/http"

	"github.com/songquanpeng/one-api/common"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type DefaultModelRetriever struct {
}

type ModelRequest struct {
	Model string `json:"model" form:"model"`
}

func (c *DefaultModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	var modelRequest ModelRequest
	e := common.UnmarshalBodyReusable(context.SrcContext, &modelRequest)
	if e != nil {
		return "", relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, e.Error(), e.Error())
	}
	return modelRequest.Model, nil
}
