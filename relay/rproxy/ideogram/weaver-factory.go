package ideogram

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

type IdeoGramWeaverFactory struct {
}

func (f *IdeoGramWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	logger.SysLogf("IdeoGramWeaverFactory start")
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&IdeoGramTokenRetriever{}).
		ModelRetriever(&IdeoGramModelRetriever{}).
		Build()
	return
}

type IdeoGramRemixWeaverFactory struct {
}

func (f *IdeoGramRemixWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&IdeoGramTokenRetriever{}).
		ModelRetriever(&IdeoGramRemixModelRetriever{}).
		Build()
	return
}

type IdeoGramPathWeaverFactory struct {
}

func (f *IdeoGramPathWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&IdeoGramTokenRetriever{}).
		ModelRetriever(&IdeoGramPathModelRetriever{}).
		Build()
	return
}

type IdeoGramV3WeaverFactory struct {
}

func (f *IdeoGramV3WeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&IdeoGramTokenRetriever{}).
		ModelRetriever(&IdeoGramV3ModelRetriever{}).
		Build()
	return
}
