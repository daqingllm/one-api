package common

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type NopTokenRetriever struct {
}

func (r *NopTokenRetriever) Retrieve(context *rproxy.RproxyContext) (token *model.Token, err *relaymodel.ErrorWithStatusCode) {
	return nil, nil
}
