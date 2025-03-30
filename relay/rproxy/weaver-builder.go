package rproxy

import "github.com/gin-gonic/gin"

type WeaverBuilder struct {
	ctx                *gin.Context
	authChecker        AuthChecker
	faultTolerancer    FaultTolerancer
	contextInitializer ContextInitializer
}

func NewWeaverBuilder(ctx *gin.Context) *WeaverBuilder {
	return &WeaverBuilder{}
}
func (w *WeaverBuilder) FailOverTolerancer(selector ChannelSelector, handler HandlerFunc) *WeaverBuilder {
	w.faultTolerancer = NewFailOverTolerancer(selector, handler)
	return w
}

func (w *WeaverBuilder) AuthChecker(authCheck AuthChecker) *WeaverBuilder {
	w.authChecker = authCheck
	return w
}
func (w *WeaverBuilder) Build() (weaver Weaver) {
	weaver = &DefaultWeaver{
		AuthChecker:     nil,
		FaultTolerancer: nil,
		RproxyAdaptor:   nil,
	}
	return
}
