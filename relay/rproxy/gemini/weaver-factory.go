package gemini

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

type GeminiGenerateWeaverFactory struct {
}

func (f *GeminiGenerateWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&GeminiTokenRetriever{}).
		ModelRetriever(&GeminiModelRetriever{}).
		Build()
	return
}
