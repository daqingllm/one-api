package rproxy

import (
	"net/http"

	"github.com/songquanpeng/one-api/relay/model"
)

type Weaver interface {
	Weave() (err *model.ErrorWithStatusCode)
	GetTokenRetriever() TokenRetriever
	GetContextInitializer() ContextInitializer
	GetFaultTolerancer() FaultTolerancer
	GetChannelSelector() ChannelSelector
}

type DefaultWeaver struct {
	FaultTolerancer    FaultTolerancer
	ContextInitializer ContextInitializer
	BillingCalculator  BillingCalculator
	ValidatorChain     *ValidatorChain
	RproxyContext      *RproxyContext
	TokenRetriever     TokenRetriever
	ModelRetriever     ModelRetriever
}

func (w *DefaultWeaver) Weave() (err *model.ErrorWithStatusCode) {
	if w.ContextInitializer == nil || w.TokenRetriever == nil {
		return model.NewErrorWithStatusCode(http.StatusInternalServerError, "weaver_struct_error", "weaver struct error ")
	}
	if err := w.ContextInitializer.Initialize(w.RproxyContext); err != nil {
		return err
	}
	if w.ValidatorChain != nil {
		if err := w.ValidatorChain.Validate(); err != nil {
			return err
		}
	}
	if err := w.FaultTolerancer.FaultTolerance(w.RproxyContext); err != nil {
		return err
	}
	return nil
}

func (w *DefaultWeaver) GetTokenRetriever() TokenRetriever {
	return w.TokenRetriever
}

func (w *DefaultWeaver) GetFaultTolerancer() FaultTolerancer {
	return w.FaultTolerancer
}

func (w *DefaultWeaver) GetChannelSelector() ChannelSelector {
	panic("implement me")
}

func (w *DefaultWeaver) GetContextInitializer() ContextInitializer {
	return w.ContextInitializer
}
