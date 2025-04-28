package router

import (
	"github.com/songquanpeng/one-api/controller"
	"github.com/songquanpeng/one-api/middleware"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/gemini"
	"github.com/songquanpeng/one-api/relay/rproxy/ideogram"
	"github.com/songquanpeng/one-api/relay/rproxy/oai"

	"github.com/gin-gonic/gin"
)

func SetRelayRouter(router *gin.Engine) {
	router.Use(middleware.CORS())
	router.Use(middleware.GzipDecodeMiddleware())
	// https://platform.openai.com/docs/api-reference/introduction
	modelsRouter := router.Group("/v1/models")
	modelsRouter.Use(middleware.TryTokenAuth())
	{
		modelsRouter.GET("", controller.ListModels)
		modelsRouter.GET("/:model", controller.RetrieveModel)
	}
	claudeV1Router := router.Group("/v1")
	claudeV1Router.Use(middleware.RelayPanicRecover(), middleware.TokenAuthClaude(), middleware.DistributeClaude(), middleware.RelayTime())
	{
		claudeV1Router.POST("/messages", controller.ClaudeMessages)
	}
	geminiRproxyRouter := router.Group("/gemini")
	geminiRproxyRouter.Use(middleware.RelayPanicRecover(), middleware.RelayTime())
	{
		geminiRproxyRouter.POST("/v1beta/models/:modelAction", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &gemini.GeminiGenerateWeaverFactory{}
		}))
		geminiRproxyRouter.POST("/upload/v1beta/files", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &gemini.GeminiFileWeaverFactory{}
		}))
		geminiRproxyRouter.GET("/v1beta/files/:filename", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &gemini.GeminiFileWeaverFactory{}
		}))
		geminiRproxyRouter.GET("/v1beta/files", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &gemini.GeminiFileWeaverFactory{}
		}))
		geminiRproxyRouter.DELETE("/v1beta/files/:filename", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &gemini.GeminiFileWeaverFactory{}
		}))

		// directRproxyRouter.POST("/gemini/v1beta/cachedContents", controller.RelayRProxy(func() rproxy.WeaverFactory {
		// 	return &gemini.GeminiCacheWeaverFactory{}
		// }))
		// directRproxyRouter.GET("/gemini/v1beta/cachedContents", controller.RelayRProxy(func() rproxy.WeaverFactory {
		// 	return &gemini.GeminiCacheWeaverFactory{}
		// }))
		// directRproxyRouter.GET("/gemini/v1beta/cachedContents/:cachedname", controller.RelayRProxy(func() rproxy.WeaverFactory {
		// 	return &gemini.GeminiCacheWeaverFactory{}
		// }))
		// directRproxyRouter.PATCH("/gemini/v1beta/cachedContents/:cachedname", controller.RelayRProxy(func() rproxy.WeaverFactory {
		// 	return &gemini.GeminiCacheWeaverFactory{}
		// }))
		// directRproxyRouter.DELETE("/gemini/v1beta/cachedContents/:cachedname", controller.RelayRProxy(func() rproxy.WeaverFactory {
		// 	return &gemini.GeminiCacheWeaverFactory{}
		// }))
		//vertex
		geminiRproxyRouter.POST("/gemini/v1/projects/:VertexAIProjectID/locations/:region/publishers/google/models/:modelAction", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &gemini.VertexGenerateWeaverFactory{}
		}))
	}
	oaiResponseRproxyRouter := router.Group("/v1")
	oaiResponseRproxyRouter.Use(middleware.RelayPanicRecover(), middleware.RelayTime())
	{
		oaiResponseRproxyRouter.POST("/responses", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &oai.OAIResponseWeaverFactory{}
		}))

		oaiResponseRproxyRouter.GET("/responses/:response_id", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &oai.OAIGetInfoWeaverFactory{}
		}))
		oaiResponseRproxyRouter.DELETE("/responses/:response_id", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &oai.OAIGetInfoWeaverFactory{}
		}))
		oaiResponseRproxyRouter.GET("/responses/:response_id/input_items", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &oai.OAIGetInfoWeaverFactory{}
		}))

	}
	ideogramRproxyRouter := router.Group("/ideogram")
	ideogramRproxyRouter.Use(middleware.RelayPanicRecover(), middleware.RelayTime())
	{
		ideogramRproxyRouter.POST("/generate", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &ideogram.IdeoGramWeaverFactory{}
		}))
		ideogramRproxyRouter.POST("/edit", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &ideogram.IdeoGramWeaverFactory{}
		}))
		ideogramRproxyRouter.POST("/remix", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &ideogram.IdeoGramRemixWeaverFactory{}
		}))
		ideogramRproxyRouter.POST("/upscale", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &ideogram.IdeoGramPathWeaverFactory{}
		}))
		ideogramRproxyRouter.POST("/describe", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &ideogram.IdeoGramPathWeaverFactory{}
		}))
		ideogramRproxyRouter.POST("/reframe", controller.RelayRProxy(func() rproxy.WeaverFactory {
			return &ideogram.IdeoGramWeaverFactory{}
		}))
	}

	relayV1Router := router.Group("/v1")
	relayV1Router.Use(middleware.RelayPanicRecover(), middleware.TokenAuth(), middleware.Distribute(), middleware.RelayTime())
	{
		relayV1Router.Any("/proxy/:channelid/*target", controller.Relay)
		relayV1Router.POST("/completions", controller.Relay)
		relayV1Router.POST("/chat/completions", controller.Relay)
		relayV1Router.POST("/edits", controller.Relay)
		relayV1Router.POST("/images/generations", controller.Relay)
		relayV1Router.POST("/images/edits", controller.Relay)
		relayV1Router.POST("/images/variations", controller.Relay)
		relayV1Router.POST("/embeddings", controller.Relay)
		relayV1Router.POST("/engines/:model/embeddings", controller.Relay)
		relayV1Router.POST("/audio/transcriptions", controller.Relay)
		relayV1Router.POST("/audio/translations", controller.Relay)
		relayV1Router.POST("/audio/speech", controller.Relay)
		relayV1Router.GET("/files", controller.RelayNotImplemented)
		relayV1Router.POST("/files", controller.RelayNotImplemented)
		relayV1Router.DELETE("/files/:id", controller.RelayNotImplemented)
		relayV1Router.GET("/files/:id", controller.RelayNotImplemented)
		relayV1Router.GET("/files/:id/content", controller.RelayNotImplemented)
		relayV1Router.POST("/fine_tuning/jobs", controller.RelayNotImplemented)
		relayV1Router.GET("/fine_tuning/jobs", controller.RelayNotImplemented)
		relayV1Router.GET("/fine_tuning/jobs/:id", controller.RelayNotImplemented)
		relayV1Router.POST("/fine_tuning/jobs/:id/cancel", controller.RelayNotImplemented)
		relayV1Router.GET("/fine_tuning/jobs/:id/events", controller.RelayNotImplemented)
		relayV1Router.DELETE("/models/:model", controller.RelayNotImplemented)
		relayV1Router.POST("/moderations", controller.Relay)
		relayV1Router.POST("/rerank", controller.Relay)
		relayV1Router.POST("/assistants", controller.RelayNotImplemented)
		relayV1Router.GET("/assistants/:id", controller.RelayNotImplemented)
		relayV1Router.POST("/assistants/:id", controller.RelayNotImplemented)
		relayV1Router.DELETE("/assistants/:id", controller.RelayNotImplemented)
		relayV1Router.GET("/assistants", controller.RelayNotImplemented)
		relayV1Router.POST("/assistants/:id/files", controller.RelayNotImplemented)
		relayV1Router.GET("/assistants/:id/files/:fileId", controller.RelayNotImplemented)
		relayV1Router.DELETE("/assistants/:id/files/:fileId", controller.RelayNotImplemented)
		relayV1Router.GET("/assistants/:id/files", controller.RelayNotImplemented)
		relayV1Router.POST("/threads", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id", controller.RelayNotImplemented)
		relayV1Router.POST("/threads/:id", controller.RelayNotImplemented)
		relayV1Router.DELETE("/threads/:id", controller.RelayNotImplemented)
		relayV1Router.POST("/threads/:id/messages", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id/messages/:messageId", controller.RelayNotImplemented)
		relayV1Router.POST("/threads/:id/messages/:messageId", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id/messages/:messageId/files/:filesId", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id/messages/:messageId/files", controller.RelayNotImplemented)
		relayV1Router.POST("/threads/:id/runs", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id/runs/:runsId", controller.RelayNotImplemented)
		relayV1Router.POST("/threads/:id/runs/:runsId", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id/runs", controller.RelayNotImplemented)
		relayV1Router.POST("/threads/:id/runs/:runsId/submit_tool_outputs", controller.RelayNotImplemented)
		relayV1Router.POST("/threads/:id/runs/:runsId/cancel", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id/runs/:runsId/steps/:stepId", controller.RelayNotImplemented)
		relayV1Router.GET("/threads/:id/runs/:runsId/steps", controller.RelayNotImplemented)
	}
}
