package rproxy

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type Handler interface {
	Handle(channel *model.Channel, context *RproxyContext) (err *relaymodel.ErrorWithStatusCode)
}
type FaultTolerancer interface {
	FaultTolerance(context *RproxyContext) (err *relaymodel.ErrorWithStatusCode)
	GetHandler() Handler
}
