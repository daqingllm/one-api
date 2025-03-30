package ideogram

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type IdeoGramWeaverFactory struct {
}

func (f *IdeoGramWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = rproxy.NewWeaverBuilder(ctx).Build()
	return
}
