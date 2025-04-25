package common

import (
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type SpecificNameModelRetriever struct {
	ModelName string
}

func (r *SpecificNameModelRetriever) Retrieve(context *rproxy.RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode) {
	return r.ModelName, nil
}
