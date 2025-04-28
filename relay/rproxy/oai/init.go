package oai

import (
	"strconv"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

func init() {
	//url-channeltype
	registry := rproxy.GetChannelAdaptorRegistry()
	var adaptorBuilder = common.DefaultHttpAdaptorBuilder{
		SetHeaderFunc:         SetHeaderFunc,
		PreCalcStrategyFunc:   PreCalcStrategyFunc,
		PostCalcStrategyFunc:  PostCalcStrategyFunc,
		ReplaceBodyParamsFunc: ReplaceBodyParamsFunc,
		GetUrlFunc:            GetUrlFunc,
		PostErrorHandleFunc:   PostErrorHandleFunc,
	}

	var nopBillingAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		GetBillingCalculator: func() rproxy.BillingCalculator {
			return &common.NOPBillingCalculator{}
		},
		SetHeaderFunc: SetHeaderFunc,
	}
	channelTypes := []string{strconv.Itoa(channeltype.OpenAI), strconv.Itoa(channeltype.Azure)}
	logger.SysLogf("register openai response channel type start %v", channelTypes)
	registry.RegisterForChannelTypes("/v1/responses", "POST", channelTypes, adaptorBuilder)
	registry.RegisterForChannelTypes("/v1/responses/:response_id", "GET", channelTypes, nopBillingAdaptorBuilder)
	registry.RegisterForChannelTypes("/v1/responses/:response_id", "DELETE", channelTypes, nopBillingAdaptorBuilder)
	registry.RegisterForChannelTypes("/v1/responses/:response_id/input_items", "GET", channelTypes, nopBillingAdaptorBuilder)
	logger.SysLogf("register openai response channel type end %v", channelTypes)

}
