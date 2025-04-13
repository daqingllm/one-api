package common

import (
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type NopModelRetriever struct {
}

func (r *NopModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	return "", nil
}
