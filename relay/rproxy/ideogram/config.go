package ideogram

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
	"github.com/tidwall/gjson"
)

var abilityChannelModelPrices = map[string]float64{
	"generate-ideogram-V_1":                  0.06 * ratio.USD,
	"generate-ideogram-V_1_TURBO":            0.02 * ratio.USD,
	"generate-ideogram-V_2":                  0.08 * ratio.USD,
	"generate-ideogram-V_2_TURBO":            0.05 * ratio.USD,
	"generate-ideogram-V_2A":                 0.04 * ratio.USD,
	"generate-ideogram-V_2A_TURBO":           0.025 * ratio.USD,
	"edit-ideogram-V_2":                      0.08 * ratio.USD,
	"edit-ideogram-V_2_TURBO":                0.05 * ratio.USD,
	"remix-ideogram-V_1":                     0.06 * ratio.USD,
	"remix-ideogram-V_1_TURBO":               0.02 * ratio.USD,
	"remix-ideogram-V_2":                     0.08 * ratio.USD,
	"remix-ideogram-V_2_TURBO":               0.05 * ratio.USD,
	"remix-ideogram-V_2A":                    0.04 * ratio.USD,
	"remix-ideogram-V_2A_TURBO":              0.025 * ratio.USD,
	"reframe-ideogram-REFRAME":               0.01 * ratio.USD,
	"upscale-ideogram-UPSCALE":               0.06 * ratio.USD,
	"describe-ideogram-DESCRIBE":             0.01 * ratio.USD,
	"generate-ideogram-V3_DEFAULT":           0.06 * ratio.USD,
	"edit-ideogram-V3_DEFAULT":               0.06 * ratio.USD,
	"remix-ideogram-V3_DEFAULT":              0.06 * ratio.USD,
	"reframe-ideogram-V3_DEFAULT":            0.06 * ratio.USD,
	"replace-background-ideogram-V3_DEFAULT": 0.06 * ratio.USD,
	"generate-ideogram-V3_TURBO":             0.03 * ratio.USD,
	"edit-ideogram-V3_TURBO":                 0.03 * ratio.USD,
	"remix-ideogram-V3_TURBO":                0.03 * ratio.USD,
	"reframe-ideogram-V3_TURBO":              0.03 * ratio.USD,
	"replace-background-ideogram-V3_TURBO":   0.03 * ratio.USD,
	"generate-ideogram-V3_QUALITY":           0.09 * ratio.USD,
	"edit-ideogram-V3_QUALITY":               0.09 * ratio.USD,
	"remix-ideogram-V3_QUALITY":              0.09 * ratio.USD,
	"reframe-ideogram-V3_QUALITY":            0.09 * ratio.USD,
	"replace-background-ideogram-V3_QUALITY": 0.09 * ratio.USD,
}

func SetHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	request.Header.Set("Api-Key", channel.Key)
	return nil
}

func GetUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	return *channel.BaseURL + strings.TrimPrefix(context.SrcContext.Request.URL.Path, "/ideogram"), nil

}
func PreV3CalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, bill *common.Bill) (err *relaymodel.ErrorWithStatusCode) {

	path := context.SrcContext.Request.URL.Path
	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	if len(parts) == 0 {
		return relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "invalid_path", "invalid_path")

	}
	lastSegment := parts[len(parts)-1]
	var renderModel string
	var batchNums float64
	contentType := context.SrcContext.Request.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if context.SrcContext.Request.MultipartForm == nil {
			err := context.SrcContext.Request.ParseMultipartForm(32 << 20)
			if err != nil {
				return relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "invalid_form_request", "invalid_form_request")
			}
		}
		renderModel = context.SrcContext.Request.Form.Get("rendering_speed")
		if renderModel == "" {
			renderModel = "DEFAULT"
		}
		batchNumsStr := context.SrcContext.Request.Form.Get("num_images")
		if batchNumsStr == "" {
			batchNums = 1
		} else {
			var e error
			batchNums, e = strconv.ParseFloat(batchNumsStr, 64)
			if e != nil {
				return relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "invalid_num_images", "invalid_num_images")
			}
		}
	}
	modelName := context.Meta.OriginModelName + "_" + renderModel
	// 检查价格是否存在
	priceKey := strings.Join([]string{lastSegment, "ideogram", modelName}, "-")
	price, exists := abilityChannelModelPrices[priceKey]
	if !exists {
		return relaymodel.NewErrorWithStatusCode(
			http.StatusBadRequest,
			"unsupported_model or rendering_speed",
			"unsupported_model or rendering_speed",
		)
	}
	// 计算价格
	var quantity float64 = price * 1000 * batchNums
	bill.PreBillItems = append(bill.PreBillItems, &common.BillItem{
		ID:        0,
		Name:      "PromptTokens",
		Quantity:  quantity,
		UnitPrice: 1,
		Quota:     int64(quantity * 1),
	})
	return nil
}

func PreCalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, bill *common.Bill) (err *relaymodel.ErrorWithStatusCode) {
	path := context.SrcContext.Request.URL.Path
	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	if len(parts) == 0 {
		return relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "invalid_path", "invalid_path")

	}
	lastSegment := parts[len(parts)-1]
	batchNums, e := getPicNums(context)
	if e != nil {
		return e
	}
	var quantity float64 = abilityChannelModelPrices[strings.Join([]string{lastSegment, "ideogram", context.Meta.OriginModelName}, "-")] * 1000 * batchNums
	bill.PreBillItems = append(bill.PreBillItems, &common.BillItem{
		ID:        0,
		Name:      "PromptTokens",
		Quantity:  quantity,
		UnitPrice: 1,
		Quota:     int64(quantity * 1),
	})
	return nil
}

func getPicNums(context *rproxy.RproxyContext) (picNum float64, err *relaymodel.ErrorWithStatusCode) {
	picNums := 1
	srcCtx := context.SrcContext
	contentType := srcCtx.Request.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		bodyBytes, err := io.ReadAll(srcCtx.Request.Body)
		if err != nil {
			return 0, relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "read_body_failed", "read_body_failed")
		}
		srcCtx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		numImagesStr := gjson.GetBytes(bodyBytes, "image_request.num_images").String()
		if numImagesStr == "" {
			return 1, nil
		}
		numImages, err := strconv.Atoi(numImagesStr)
		if err != nil {
			return 1, &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusBadRequest,
				Error:      relaymodel.Error{Message: "invalid_num_images", Code: "INVALID_NUM_IMAGES"},
			}
		}
		picNums = numImages
		return float64(picNums), nil

	} else if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if srcCtx.Request.MultipartForm == nil {
			err := srcCtx.Request.ParseMultipartForm(32 << 20)
			if err != nil {
				return 1, &relaymodel.ErrorWithStatusCode{
					StatusCode: http.StatusBadRequest,
					Error:      relaymodel.Error{Message: "invalid_form_request", Code: "INVALID_FORM"},
				}
			}
		}
		numImagesStr := srcCtx.Request.Form.Get("num_images")
		if numImagesStr == "" {
			imageRequestStr := srcCtx.Request.Form.Get("image_request")
			if imageRequestStr != "" {
				numImagesStr = gjson.Get(imageRequestStr, "num_images").String()
			}
		}
		if numImagesStr == "" {
			return 1, nil
		}
		numImages, err := strconv.Atoi(numImagesStr)
		if err != nil {
			return 1, &relaymodel.ErrorWithStatusCode{
				StatusCode: http.StatusBadRequest,
				Error:      relaymodel.Error{Message: "invalid_num_images", Code: "INVALID_NUM_IMAGES"},
			}
		}
		picNum = float64(numImages)
		return picNum, nil
	}
	return 1, nil
}
