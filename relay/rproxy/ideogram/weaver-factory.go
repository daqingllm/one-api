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
