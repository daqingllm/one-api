package oai

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

type OAIResponseWeaverFactory struct {
}

func (f *OAIResponseWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&OAITokenRetriever{}).
		ModelRetriever(&OAIModelRetriever{}).
		Build()
	return
}

type OAIGetInfoWeaverFactory struct {
}

func (f *OAIGetInfoWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&OAITokenRetriever{}).
		ModelRetriever(&common.NopModelRetriever{}).
		Build()
	return
}
