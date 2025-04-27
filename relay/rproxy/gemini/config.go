package gemini

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/adaptor/vertexai"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
	"github.com/songquanpeng/one-api/relay/util"
	"github.com/tidwall/gjson"
)

func GetUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	var baseURL string = *channel.BaseURL
	var basePath string = strings.TrimPrefix(context.SrcContext.Request.URL.Path, "/gemini")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	rawQuery := context.SrcContext.Request.URL.RawQuery

	if rawQuery != "" {
		queryParams := context.SrcContext.Request.URL.Query()
		queryParams.Del("key")
		newRawQuery := queryParams.Encode()
		if newRawQuery != "" {
			return baseURL + basePath + "?" + newRawQuery + "&key=" + channel.Key, nil
		}
	}
	return baseURL + basePath + "?key=" + channel.Key, nil

}

func GetFileUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	var baseURL string = *channel.BaseURL
	var basePath string = strings.TrimPrefix(context.SrcContext.Request.URL.Path, "/gemini")
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	rawQuery := context.SrcContext.Request.URL.RawQuery
	if rawQuery != "" {
		return baseURL + basePath + "?" + rawQuery, nil
	}
	return baseURL + basePath, nil

}

func GetVertexUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	// 检查请求路径是否包含/v1beta/models，如果是原生gemini，需要转换为vertex格式的请求路径
	var basePath string = strings.TrimPrefix(context.SrcContext.Request.URL.Path, "/gemini")
	if strings.Contains(context.SrcContext.Request.URL.Path, "/v1beta/models") {
		modelAction := context.SrcContext.Param("modelAction")
		config, err := channel.LoadConfig()
		if err != nil {
			return "", &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusInternalServerError,
				Error:      relaymodel.Error{Message: "load_config_failed", Code: "LOAD_CONFIG_FAILED"},
			}
		}
		region := config.Region
		if region == "" {
			region = "us-central1"
		}
		projectID := config.VertexAIProjectID
		if projectID == "" {
			return "", &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusInternalServerError,
				Error:      relaymodel.Error{Message: "missing_project_id", Code: "MISSING_PROJECT_ID"},
			}
		}
		newPath := fmt.Sprintf("/v1/projects/%s/locations/%s/publishers/google/models/%s",
			projectID, region, modelAction)

		context.SrcContext.Request.URL.Path = newPath
		logger.Infof(context.SrcContext, "转换Gemini路径为Vertex AI格式: %s", newPath)
	}
	var baseURL string = *channel.BaseURL
	if baseURL == "" {
		config, err := channel.LoadConfig()
		if err != nil {
			return "", &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusInternalServerError,
				Error:      relaymodel.Error{Message: "load_config_failed", Code: "LOAD_CONFIG_FAILED"},
			}
		}
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com", config.Region)
	}
	rawQuery := context.SrcContext.Request.URL.RawQuery
	if rawQuery != "" {
		return baseURL + basePath + "?" + rawQuery, nil
	}
	return baseURL + basePath, nil

}

func SetVertexHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	config, e := channel.LoadConfig()
	if e != nil {
		return &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusInternalServerError,
			Error:      relaymodel.Error{Message: "load_config_failed", Code: "LOAD_CONFIG_FAILED"},
		}
	}
	token, e := vertexai.GetToken(context.SrcContext, channel.Id, config.VertexAIADC)
	if e != nil {
		return &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusInternalServerError,
			Error:      relaymodel.Error{Message: "get_token_failed", Code: "GET_TOKEN_FAILED"},
		}
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func SetFileHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	request.Header.Set("x-goog-api-key", channel.Key)
	return nil
}

func PreCalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, bill *common.Bill) (err *relaymodel.ErrorWithStatusCode) {
	parsed := gjson.ParseBytes(context.ResolvedRequest.([]byte))
	input := parsed.Get("contents").String()
	promptTokens := int(config.PreConsumedQuota) + openai.CountTokenInput(input, context.GetRequestModel())

	maxTokens := parsed.Get("generationConfig.maxOutputTokens").Int()
	if maxTokens != 0 {
		promptTokens += int(maxTokens)

	}
	bill.PreBillItems = append(bill.PreBillItems, common.TokenUsageBillItem(common.PromptTokens, 1, float64(promptTokens)))
	//web search
	tools := parsed.Get("tools").Array()
	for _, tool := range tools {
		if tool.Get("googleSearch").Exists() {
			bill.PreBillItems = append(bill.PreBillItems, common.PayperUseBillItem(common.WebSearch, 0.035, 1))
			break
		}
	}
	return nil
}

func CachePostCalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, bill *common.Bill) (err *relaymodel.ErrorWithStatusCode) {
	// parsed := gjson.ParseBytes(context.ResolvedRequest.([]byte))
	// input := parsed.Get("contents").String()
	// promptTokens := int(config.PreConsumedQuota) + openai.CountTokenInput(input, context.GetRequestModel())

	// maxTokens := parsed.Get("generationConfig.maxOutputTokens").Int()
	// if maxTokens != 0 {
	// 	promptTokens += int(maxTokens)
	// }
	// bill.PreBillItems = append(bill.PreBillItems, common.TokenUsageBillItem(common.PromptTokens, 1, float64(promptTokens)))
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
		if parsed.IsArray() {
			lastResponse := parsed.Array()[len(parsed.Array())-1]
			if usage := lastResponse.Get("usageMetadata"); usage.Exists() {
				totalUsage.InputTokens += int(usage.Get("promptTokenCount").Int())
				totalUsage.OutputTokens += int(usage.Get("candidatesTokenCount").Int())
			}
		} else {
			if usage := parsed.Get("usageMetadata"); usage.Exists() {
				totalUsage.InputTokens += int(usage.Get("promptTokenCount").Int())
				totalUsage.OutputTokens += int(usage.Get("candidatesTokenCount").Int())
			}
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
		bill.BillItems = append(bill.BillItems, common.TokenUsageBillItem(common.PromptTokens, 1, float64(totalUsage.InputTokens)))
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

func StreamHandFunc(context *rproxy.RproxyContext, resp rproxy.Response) (result any, err *relaymodel.ErrorWithStatusCode) {
	query := context.SrcContext.Request.URL.Query()
	if query.Get("alt") == "sse" {
		return util.StreamGeminiSSEResponseHandle(context.SrcContext, resp.(*http.Response))
	}
	return util.StreamGeminiResponseHandle(context.SrcContext, resp.(*http.Response))
}

func PostInitializeFunc(context *rproxy.RproxyContext) *relaymodel.ErrorWithStatusCode {
	if strings.HasSuffix(context.SrcContext.Request.URL.Path, "streamGenerateContent") {
		context.Meta.IsStream = true
	}
	srcCtx := context.SrcContext
	contentType := srcCtx.Request.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		bodyBytes, err := io.ReadAll(srcCtx.Request.Body)
		if err != nil {
			return &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusBadRequest,
				Error:      relaymodel.Error{Message: "read_request_body_failed", Code: "READ_BODY_FAILED"},
			}
		}
		srcCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		context.ResolvedRequest = bodyBytes
	}
	return nil
}
