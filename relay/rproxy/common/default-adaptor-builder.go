package common

import (
	"net/http"

	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type DefaultHttpAdaptorBuilder struct {
	GetErrorHandler       func() rproxy.ErrorHandler
	GetRequestHandler     func() rproxy.RequestHandler
	GetResponseHandler    func() rproxy.ResponseHandler
	GetBillingCalculator  func() rproxy.BillingCalculator
	GetUrlFunc            func(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode)
	SetHeaderFunc         func(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode)
	ReplaceBodyParamsFunc func(context *rproxy.RproxyContext, channel *model.Channel, body []byte) (replacedBody []byte, err *relaymodel.ErrorWithStatusCode)

	// CalcStrategyFunc      func(context *rproxy.RproxyContext, channel *model.Channel, groupRatio float64) (preConsumedQuota int64, err *relaymodel.ErrorWithStatusCode)
	PreCalcStrategyFunc   func(context *rproxy.RproxyContext, channel *model.Channel, bill *Bill) (err *relaymodel.ErrorWithStatusCode)
	PostCalcStrategyFunc  func(context *rproxy.RproxyContext, channel *model.Channel, bill *Bill) (err *relaymodel.ErrorWithStatusCode)
	FinalCalcStrategyFunc func(context *rproxy.RproxyContext, channel *model.Channel, bill *Bill) (err *relaymodel.ErrorWithStatusCode)
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
	var replaceBodyParamsFunc func(context *rproxy.RproxyContext, channel *model.Channel, body []byte) (replacedBody []byte, err *relaymodel.ErrorWithStatusCode)

	if b.ReplaceBodyParamsFunc != nil {
		replaceBodyParamsFunc = b.ReplaceBodyParamsFunc
	}
	var errorHandler rproxy.ErrorHandler
	if b.GetErrorHandler == nil || b.GetErrorHandler() == nil {
		errorHandler = &DefaultErrorHandler{}
	} else {
		errorHandler = b.GetErrorHandler()
	}
	var requestHandler rproxy.RequestHandler
	if b.GetRequestHandler == nil || b.GetRequestHandler() == nil {
		requestHandler = &DefaultRequestHandler{
			GetUrlFunc:            getUrlFunc,
			SetHeaderFunc:         setHeaderFunc,
			ReplaceBodyParamsFunc: replaceBodyParamsFunc,
		}
	} else {
		requestHandler = b.GetRequestHandler()
	}
	var responseHandler rproxy.ResponseHandler
	if b.GetResponseHandler == nil || b.GetResponseHandler() == nil {
		responseHandler = &DefaultResponseHandler{}
	} else {
		responseHandler = b.GetResponseHandler()
	}
	var billingCalculator rproxy.BillingCalculator
	if b.GetBillingCalculator == nil {
		billingCalculator = &DefaultBillingCalculator{
			PreCalcStrategyFunc:   b.PreCalcStrategyFunc,
			PostCalcStrategyFunc:  b.PostCalcStrategyFunc,
			FinalCalcStrategyFunc: b.FinalCalcStrategyFunc,
		}
	} else {
		billingCalculator = b.GetBillingCalculator()
	}

	adaptor = &HttpRproxyAdaptor{
		Errorhandler:      errorHandler,
		RequestHandler:    requestHandler,
		ResponseHandler:   responseHandler,
		BillingCalculator: billingCalculator,
	}
	if handler, ok := adaptor.(*HttpRproxyAdaptor).RequestHandler.(*DefaultRequestHandler); ok {
		handler.Adaptor = adaptor

	}
	if calc, ok := adaptor.(*HttpRproxyAdaptor).BillingCalculator.(*DefaultBillingCalculator); ok {
		calc.adaptor = adaptor
	}
	return
}
