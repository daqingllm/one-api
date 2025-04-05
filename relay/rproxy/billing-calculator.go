package rproxy

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

type BillingCalculator interface {
	GetChannel() *model.Channel
	PreCalAndExecute(context *RproxyContext) *relaymodel.ErrorWithStatusCode
	RollBackPreCalAndExecute(context *RproxyContext) *relaymodel.ErrorWithStatusCode
	PostCalcAndExecute(context *RproxyContext) *relaymodel.ErrorWithStatusCode
}
