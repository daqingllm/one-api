package rproxy

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type ErrorHandler interface {
	HandleError(context *RproxyContext, resp Response, e error) (err *relaymodel.ErrorWithStatusCode)
}

type RequestHandler interface {
	Handle(context *RproxyContext) (req Request, err *relaymodel.ErrorWithStatusCode)
}

type ResponseHandler interface {
	Handle(context *RproxyContext, resp Response) (err *relaymodel.ErrorWithStatusCode)
}

type Request interface{}
type Response interface{}
type RproxyAdaptor interface {
	GetChannel() *model.Channel
	SetChannel(channel *model.Channel)
	GetRequestHandler() RequestHandler
	DoRequest(context *RproxyContext) (response Response, err *relaymodel.ErrorWithStatusCode)
	GetResponseHandler() ResponseHandler
	GetErrorHandler() ErrorHandler
}
