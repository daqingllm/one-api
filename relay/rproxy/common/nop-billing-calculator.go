package common

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type NopBillingCalculator struct {
}

func (b *NopBillingCalculator) GetChannel() *model.Channel {
	return nil
}

func (b *NopBillingCalculator) PreCalAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	return nil
}

func (b *NopBillingCalculator) RollBackPreCalAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	return nil
}

func (b *NopBillingCalculator) PostCalcAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	return nil
}
