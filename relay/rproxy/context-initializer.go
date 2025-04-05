package rproxy

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type ContextInitializer interface {
	Initialize(context *RproxyContext) *relaymodel.ErrorWithStatusCode
	GetTokenRetriever() TokenRetriever
	GetModelRetriever() ModelRetriever
}

type TokenRetriever interface {
	Retrieve(context *RproxyContext) (token *model.Token, err *relaymodel.ErrorWithStatusCode)
}

type ModelRetriever interface {
	Retrieve(context *RproxyContext) (modelName string, err *relaymodel.ErrorWithStatusCode)
}
