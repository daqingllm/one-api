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
		PostInitializeFunc(PostInitializeFunc).
		Build()
	return
}

type VertexGenerateWeaverFactory struct {
}

func (f *VertexGenerateWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&common.AuthorizationTokenRetriever{}).
		ModelRetriever(&GeminiModelRetriever{}).
		PostInitializeFunc(PostInitializeFunc).
		Build()
	return
}

type GeminiFileWeaverFactory struct {
}

func (f *GeminiFileWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&GeminiTokenRetriever{}).
		ModelRetriever(&common.SpecificNameModelRetriever{ModelName: "gemini-2.5-pro-preview-03-25"}).
		Build()
	return
}

type GeminiCacheWeaverFactory struct {
}

func (f *GeminiCacheWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	weaver = common.
		NewWeaverBuilder(ctx).
		TokenRetriever(&GeminiTokenRetriever{}).
		ModelRetriever(&GeminiCacheModelRetriever{}).
		Build()
	return
}
