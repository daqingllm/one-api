package common

import (
	"github.com/songquanpeng/one-api/model"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type NOPBillingCalculator struct {
}

func (b *NOPBillingCalculator) GetChannel() *model.Channel {
	return nil
}

func (b *NOPBillingCalculator) PreCalAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	return nil
}

func (b *NOPBillingCalculator) RollBackPreCalAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	return nil
}

func (b *NOPBillingCalculator) PostCalcAndExecute(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	return nil
}
