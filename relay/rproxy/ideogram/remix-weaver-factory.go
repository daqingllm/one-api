package ideogram

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

type IdeoGramRemixWeaverFactory struct {
}

func (f *IdeoGramRemixWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&IdeoGramTokenRetriever{}).
		ModelRetriever(&IdeoGramEditModelRetriever{}).
		Build()
	return
}
