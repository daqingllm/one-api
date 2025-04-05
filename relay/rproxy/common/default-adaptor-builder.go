package common

import (
	"net/http"

	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type DefaultHttpAdaptorBuilder struct {
	GetUrlFunc       func(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode)
	SetHeaderFunc    func(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode)
	CalcStrategyFunc func(context *rproxy.RproxyContext, ratio float64) (preConsumedQuota int64, err *relaymodel.ErrorWithStatusCode)
}

func SetNopHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	// do nothing
	return nil
}

func GetUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	return *channel.BaseURL + context.SrcContext.Request.URL.Path, nil

}
func (b DefaultHttpAdaptorBuilder) Build() (adaptor rproxy.RproxyAdaptor) {
	var getUrlFunc func(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) = nil
	if b.GetUrlFunc != nil {
		getUrlFunc = b.GetUrlFunc
	} else {
		getUrlFunc = GetUrlFunc
	}
	var setHeaderFunc func(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) = nil
	if b.SetHeaderFunc != nil {
		setHeaderFunc = b.SetHeaderFunc

	} else {
		setHeaderFunc = SetNopHeaderFunc
	}
	billingCalculator := &DefaultBillingCalculator{
		CalcStrategyFunc: b.CalcStrategyFunc,
	}
	adaptor = &HttpRproxyAdaptor{
		Errorhandler: &DefaultErrorHandler{},
		RequestHandler: &DefaultRequestHandler{
			GetUrlFunc:    getUrlFunc,
			SetHeaderFunc: setHeaderFunc,
		},
		ResponseHandler:   &DefaultResponseHandler{},
		BillingCalculator: billingCalculator,
	}
	if handler, ok := adaptor.(*HttpRproxyAdaptor).RequestHandler.(*DefaultRequestHandler); ok {
		handler.Adaptor = adaptor
		billingCalculator.adaptor = adaptor
	}
	return
}
