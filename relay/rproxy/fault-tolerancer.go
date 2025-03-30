package rproxy

import "github.com/songquanpeng/one-api/relay/model"

type FaultTolerancer interface {
	FaultTolerance(context *RproxyContext) (err *model.ErrorWithStatusCode)
}
