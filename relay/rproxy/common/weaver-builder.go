package common

import (
	"github.com/gin-gonic/gin"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type WeaverBuilder struct {
	context            *rproxy.RproxyContext
	faultTolerancer    rproxy.FaultTolerancer
	contextInitializer rproxy.ContextInitializer
	validators         []rproxy.Validator
	tokenRetrierver    rproxy.TokenRetriever
	modelRetriever     rproxy.ModelRetriever
	postInitializeFunc func(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode
}

func NewWeaverBuilder(ctx *gin.Context) *WeaverBuilder {
	return &WeaverBuilder{
		context: &rproxy.RproxyContext{
			SrcContext: ctx,
		},
	}
}
func (w *WeaverBuilder) FailOverTolerancer(selector rproxy.ChannelSelector, handler rproxy.Handler) *WeaverBuilder {
	w.faultTolerancer = rproxy.NewFailOverTolerancer(selector, nil)
	return w
}
func (w *WeaverBuilder) AddValidators(validators ...rproxy.Validator) *WeaverBuilder {
	w.validators = append(w.validators, validators...)
	return w
}

func (w *WeaverBuilder) PostInitializeFunc(postInitializeFunc func(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode) *WeaverBuilder {
	w.postInitializeFunc = postInitializeFunc
	return w
}

func (w *WeaverBuilder) ContextInitializer(contextInitializer rproxy.ContextInitializer) *WeaverBuilder {
	w.contextInitializer = contextInitializer
	return w
}

func (w *WeaverBuilder) TokenRetriever(tokenRetriever rproxy.TokenRetriever) *WeaverBuilder {
	w.tokenRetrierver = tokenRetriever
	return w
}

func (w *WeaverBuilder) ModelRetriever(modelRetriever rproxy.ModelRetriever) *WeaverBuilder {
	w.modelRetriever = modelRetriever
	return w
}

func (w *WeaverBuilder) Build() (weaver rproxy.Weaver) {
	if w.faultTolerancer == nil {
		w.faultTolerancer = rproxy.NewFailOverTolerancer(NewDefaultChannelSelector(), NewToleranceHandler())
	}
	if w.tokenRetrierver == nil || w.modelRetriever == nil {
		// 不可恢复错误，tokenRetriever为空，则抛出panic
		panic("token_and_model_retriever_not_found")
	}
	if w.contextInitializer == nil {
		w.contextInitializer = &DefaultContextInitializer{
			modelRetrierver:    w.modelRetriever,
			tokenRetrierver:    w.tokenRetrierver,
			PostInitializeFunc: w.postInitializeFunc,
		}
	}

	validatorChain := rproxy.NewValidatorChain().
		AddValidator(&TokenValidator{
			ctx: w.context,
		}).
		AddValidator(&QuotaValidator{
			ctx: w.context,
		}).
		AddValidator(&IpValidator{
			ctx: w.context,
		}).
		AddValidator(&BlacklistValidator{
			ctx: w.context,
		}).
		AddValidator(&ChannelValidator{
			ctx: w.context,
		})
	weaver = &rproxy.DefaultWeaver{
		FaultTolerancer:    w.faultTolerancer,
		ValidatorChain:     validatorChain.AddValidators(w.validators...),
		RproxyContext:      w.context,
		ContextInitializer: w.contextInitializer,
		TokenRetriever:     w.tokenRetrierver,
	}
	return
}
