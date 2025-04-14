package oai

import (
	"encoding/json"
	"fmt"
	"net/http"
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

var abilityWebSearchPrice = map[string]float64{
	"gpt-4o_low_1":                        0.03 * ratio.USD,
	"gpt-4o_medium_1":                     0.035 * ratio.USD,
	"gpt-4o_high_1":                       0.05 * ratio.USD,
	"gpt-4o-search-preview_low_1":         0.03 * ratio.USD,
	"gpt-4o-search-preview_medium_1":      0.035 * ratio.USD,
	"gpt-4o-search-preview_high_1":        0.05 * ratio.USD,
	"gpt-4o-mini_low_1":                   0.025 * ratio.USD,
	"gpt-4o-mini_medium_1":                0.0275 * ratio.USD,
	"gpt-4o-mini_high_1":                  0.03 * ratio.USD,
	"gpt-4o-mini-search-preview_low_1":    0.025 * ratio.USD,
	"gpt-4o-mini-search-preview_medium_1": 0.0275 * ratio.USD,
	"gpt-4o-mini-search-preview_high_1":   0.03 * ratio.USD,
}

func GetUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	var baseURL string = *channel.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	if channel.Type == 3 {
		modifiedPath := strings.Replace(context.SrcContext.Request.URL.Path, "/v1", "/openai", 1)
		return baseURL + modifiedPath + "?api-version=" + *channel.Other, nil
	}
	return baseURL + context.SrcContext.Request.URL.Path, nil

}

func SetHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))

	return nil
}

func PreCalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, bill *common.Bill) (err *relaymodel.ErrorWithStatusCode) {
	parsed := gjson.ParseBytes(context.ResolvedRequest.([]byte))
	input := parsed.Get("input").String()
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
	// tools := parsed.Get("tools").Array()
	// if len(tools) == 0 {
	// 	return nil
	// }
	// var webSearch bool = false
	// var fileSearch bool = false
	// for _, tool := range tools {
	// 	toolType := tool.Get("type").String() // 假设每个 tool 对象有 type 字段
	// 	switch toolType {
	// 	case "web_search_preview": // 根据实际 API 字段调整
	// 		webSearch = true
	// 	case "file_search": // 根据实际 API 字段调整
	// 		fileSearch = true
	// 	}
	// }
	// // 根据功能启用情况生成计费项（示例数值需按实际计费规则调整）
	// if webSearch {
	// 	billItem := &common.BillItem{
	// 		Name:      "WebSearchTokens",
	// 		UnitPrice: 1,             // 单价示例
	// 		Quantity:  1000,            // 按次或按 token 数计算
	// 		Quota:     int64(1000 * 1), // 配额换算系数
	// 	}
	// 	bill.PreBillItems = append(bill.PreBillItems, billItem)
	// }

	// if fileSearch {
	// 	billItem := &common.BillItem{
	// 		Name:      "FileSearchTokens",
	// 		UnitPrice: 0.5, // 单价示例
	// 		Quantity:  500, // 按次或按 token 数计算
	// 		Quota:     int64(500 * 1),
	// 	}
	// 	bill.PreBillItems = append(bill.PreBillItems, billItem)
	// }
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
		if usage := parsed.Get("response.usage"); usage.Exists() {
			totalUsage.InputTokens += int(usage.Get("input_tokens").Int())
			totalUsage.OutputTokens += int(usage.Get("output_tokens").Int())
		}
	} else {

		parsed := gjson.ParseBytes(context.ResolvedResponse.([]byte))
		if config.DebugUserIds[context.GetUserId()] {
			logger.DebugForcef(context.SrcContext, "usage:%v", parsed)
		}
		if usage := parsed.Get("usage"); usage.Exists() {
			totalUsage.InputTokens += int(usage.Get("input_tokens").Int())
			totalUsage.OutputTokens += int(usage.Get("output_tokens").Int())
			// totalUsage.CachedTokens += int(usage.Get("cached_tokens").Int())
			// totalUsage.SearchTokens += int(usage.Get("search_tokens").Int())
		}
	}

	// 创建账单明细项
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

	// 添加搜索token费用
	// if totalUsage.SearchTokens > 0 {
	// 	bill.BillItems = append(bill.BillItems, &common.BillItem{
	// 		Name:      "SearchTokens",
	// 		UnitPrice: 0.002 * ratio.USD,
	// 		Quantity:  float64(totalUsage.SearchTokens),
	// 		Quota:     int64(0.002 * ratio.USD * float64(totalUsage.SearchTokens)),
	// 	})
	// }

	// 应用缓存折扣
	// if totalUsage.CachedTokens > 0 {
	// 	bill.Discounts = append(bill.Discounts, &common.Discount{
	// 		ID:       "CacheTokens",
	// 		Name:     "流式缓存折扣",
	// 		Ratio:    0.5, // 50%折扣
	// 		Describe: fmt.Sprintf("缓存%d tokens", totalUsage.CachedTokens),
	// 	})
	// }

	return nil
}
func ReplaceBodyParamsFunc(context *rproxy.RproxyContext, channel *model.Channel, body []byte) (replacedBody []byte, err *relaymodel.ErrorWithStatusCode) {

	modelMapping := channel.GetModelMapping()
	if modelMapping == nil {
		return body, nil
	}

	actualModel := modelMapping[context.GetRequestModel()]
	if actualModel == "" {
		return body, nil
	}

	var jsonBody map[string]interface{}
	if err := json.Unmarshal(body, &jsonBody); err != nil {
		if _, ok := err.(*json.SyntaxError); ok {
			return body, nil
		}
		logger.Errorf(context.SrcContext, "JSON解析失败: %v", err)
		return nil, relaymodel.NewErrorWithStatusCode(
			http.StatusBadRequest,
			"invalid_json",
			"无效的JSON格式",
		)
	}
	jsonBody["model"] = actualModel

	modifiedBody, e := json.Marshal(jsonBody)
	if e != nil {
		logger.Errorf(context.SrcContext, "JSON序列化失败: %v", err)
		return nil, relaymodel.NewErrorWithStatusCode(
			http.StatusInternalServerError,
			"serialize_failed",
			"JSON序列化失败",
		)
	}
	return modifiedBody, nil
}

func getKey(path string, method string, channelType int) string {
	return strings.Join([]string{path, method, strconv.Itoa(channelType)}, "-")
}
func init() {
	//url-channeltype
	registry := rproxy.GetChannelAdaptorRegistry()
	var adaptorBuilder = common.DefaultHttpAdaptorBuilder{
		SetHeaderFunc:         SetHeaderFunc,
		PreCalcStrategyFunc:   PreCalcStrategyFunc,
		PostCalcStrategyFunc:  PostCalcStrategyFunc,
		ReplaceBodyParamsFunc: ReplaceBodyParamsFunc,
		GetUrlFunc:            GetUrlFunc,
	}

	var nopBillingAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		GetBillingCalculator: func() rproxy.BillingCalculator {
			return &common.NOPBillingCalculator{}
		},
		SetHeaderFunc: SetHeaderFunc,
	}
	channelTypes := []int{channeltype.OpenAI, channeltype.Azure}
	for _, channelType := range channelTypes {
		logger.SysLogf("register openai response channel type start %d", channelType)
		registry.Register(getKey("/v1/responses", "POST", channelType), adaptorBuilder)
		registry.Register(getKey("/v1/responses/:response_id", "GET", channelType), nopBillingAdaptorBuilder)
		registry.Register(getKey("/v1/responses/:response_id", "DELETE", channelType), nopBillingAdaptorBuilder)
		registry.Register(getKey("/v1/responses/:response_id/input_items", "GET", channelType), nopBillingAdaptorBuilder)
		logger.SysLogf("register openai response channel type end %d", channelType)
	}

}
