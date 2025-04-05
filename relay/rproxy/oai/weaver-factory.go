package oai

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type OAIResponseWeaverFactory struct {
}

func (f *OAIResponseWeaverFactory) GetWeaver(ctx *gin.Context) (weaver rproxy.Weaver) {
	// weaver = rproxy.NewWeaverBuilder(ctx).
	// 	FailOverTolerancer(rproxy.DefaultChannelSelector{}, func(channel *model.Channel, context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	// 		httpRproxyAdaptor := rproxy.HttpRproxyAdaptor{
	// 			GetRequestUrl: func(context *rproxy.RproxyContext) string {
	// 				return channel.GetBaseURL()
	// 			},
	// 			HandlerRequestHeader: func(context *rproxy.RproxyContext) map[string]string {
	// 				return map[string]string{
	// 					"Content-Type":  "application/json",
	// 					"Authorization": "Bearer " + channel.Key,
	// 				}
	// 			},
	// 		}
	// 		e := httpRproxyAdaptor.DoRequest(context)
	// 		if e != nil {
	// 			return e
	// 		}
	// 		return nil
	// 	}).
	// 	Build()
	return nil
}
