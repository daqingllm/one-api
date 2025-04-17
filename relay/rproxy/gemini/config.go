package gemini

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
	"github.com/tidwall/gjson"
)

func GetUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	var baseURL string = *channel.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return baseURL + context.SrcContext.Request.URL.Path + "?key=" + channel.Key, nil

}

func PreCalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, bill *common.Bill) (err *relaymodel.ErrorWithStatusCode) {
	parsed := gjson.ParseBytes(context.ResolvedRequest.([]byte))
	input := parsed.Get("contents").String()
	promptTokens := int(config.PreConsumedQuota) + openai.CountTokenInput(input, context.GetRequestModel())

	maxTokens := parsed.Get("max_output_tokens").Int()
	if maxTokens != 0 {
		promptTokens += int(maxTokens)

	}
	billItem := &common.BillItem{
		Name:      "PromptTokens",
		UnitPrice: 1,
		Quantity:  float64(promptTokens),
		Quota:     int64(float64(promptTokens) * 1),
	}
	bill.PreBillItems = append(bill.PreBillItems, billItem)
	return nil
}

func PostCalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, bill *common.Bill) (err *relaymodel.ErrorWithStatusCode) {
	var totalUsage struct {
		InputTokens         int
		InputTokensDetails  any
		OutputTokens        int
		OutputTokensDetails any
		totalTokens         int
	}

	if context.IsStream() {
		parsed := gjson.ParseBytes(context.ResolvedResponse.([]byte))
		if config.DebugUserIds[context.GetUserId()] {
			logger.DebugForcef(context.SrcContext, "usage:%s", parsed)
		}
		if usage := parsed.Get("response.usageMetadata"); usage.Exists() {
			totalUsage.InputTokens += int(usage.Get("promptTokenCount").Int())
			totalUsage.OutputTokens += int(usage.Get("candidatesTokenCount").Int())
		}
	} else {

		parsed := gjson.ParseBytes(context.ResolvedResponse.([]byte))
		if config.DebugUserIds[context.GetUserId()] {
			logger.DebugForcef(context.SrcContext, "usage:%v", parsed)
		}
		if usage := parsed.Get("usageMetadata"); usage.Exists() {
			totalUsage.InputTokens += int(usage.Get("promptTokenCount").Int())
			totalUsage.OutputTokens += int(usage.Get("candidatesTokenCount").Int())
		}
	}

	if totalUsage.InputTokens > 0 {
		bill.BillItems = append(bill.BillItems, &common.BillItem{
			Name:      "PromptTokens",
			UnitPrice: 1,
			Quantity:  float64(totalUsage.InputTokens),
			Quota:     int64(float64(totalUsage.InputTokens)),
		})
	}

	if totalUsage.OutputTokens > 0 {
		var completionRatio = ratio.GetCompletionRatio(context.GetOriginalModel(), channel.Type)
		billItem := &common.BillItem{
			Name:      "CompletionTokens",
			UnitPrice: completionRatio,
			Quantity:  float64(totalUsage.OutputTokens),
			Quota:     int64(float64(totalUsage.OutputTokens) * completionRatio),
			Discount: &common.Discount{
				ID:       "completion_ratio",
				Name:     "补全倍率",
				Type:     0, // 0 表示模型级折扣
				Ratio:    completionRatio,
				Describe: fmt.Sprintf(" %s 费率系数", context.GetOriginalModel()),
			},
		}
		bill.BillItems = append(bill.BillItems, billItem)

	}

	return nil
}

func getKey(path string, method string, channelType int) string {
	return strings.Join([]string{path, method, strconv.Itoa(channelType)}, "-")
}
func init() {
	//url-channeltype
	registry := rproxy.GetChannelAdaptorRegistry()
	var adaptorBuilder = common.DefaultHttpAdaptorBuilder{
		PreCalcStrategyFunc:  PreCalcStrategyFunc,
		PostCalcStrategyFunc: PostCalcStrategyFunc,
		GetUrlFunc:           GetUrlFunc,
	}

	logger.SysLogf("register gemin response channel type start %d", channeltype.Gemini)
	registry.Register(getKey("/v1beta/models/:model/*action", "POST", channeltype.Gemini), adaptorBuilder)
	logger.SysLogf("register gemin response channel type end %d", channeltype.Gemini)

}
