package ideogram

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/billing/ratio"
	"github.com/songquanpeng/one-api/relay/channeltype"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
	"github.com/tidwall/gjson"
)

var abilityChannelModelPrices = map[string]float64{
	"generate-ideogram-V_1":        0.06 * ratio.USD,
	"generate-ideogram-V_1_TURBO":  0.02 * ratio.USD,
	"generate-ideogram-V_2":        0.08 * ratio.USD,
	"generate-ideogram-V_2_TURBO":  0.05 * ratio.USD,
	"generate-ideogram-V_2A":       0.04 * ratio.USD,
	"generate-ideogram-V_2A_TURBO": 0.025 * ratio.USD,
	"edit-ideogram-V_2":            0.08 * ratio.USD,
	"edit-ideogram-V_2_TURBO":      0.05 * ratio.USD,
	"remix-ideogram-V_1":           0.06 * ratio.USD,
	"remix-ideogram-V_1_TURBO":     0.02 * ratio.USD,
	"remix-ideogram-V_2":           0.08 * ratio.USD,
	"remix-ideogram-V_2_TURBO":     0.05 * ratio.USD,
	"remix-ideogram-V_2A":          0.04 * ratio.USD,
	"remix-ideogram-V_2A_TURBO":    0.025 * ratio.USD,
	"reframe-ideogram-REFRAME":     0.01 * ratio.USD,
	"upscale-ideogram-UPSCALE":     0.06 * ratio.USD,
	"describe-ideogram-DESCRIBE":   0.01 * ratio.USD,
}

func SetHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	request.Header.Set("Api-Key", channel.Key)
	return nil
}

func GetUrlFunc(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode) {
	return *channel.BaseURL + strings.TrimPrefix(context.SrcContext.Request.URL.Path, "/ideogram"), nil

}

func CalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, groupRatio float64) (preConsumedQuota int64, err *relaymodel.ErrorWithStatusCode) {
	path := context.SrcContext.Request.URL.Path
	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	if len(parts) == 0 {
		return 0, relaymodel.NewErrorWithStatusCode(http.StatusBadRequest, "invalid_path", "invalid_path")

	}
	lastSegment := parts[len(parts)-1]
	batchNums, e := getPicNums(context)
	if e != nil {
		return 0, e
	}
	return int64(abilityChannelModelPrices[strings.Join([]string{lastSegment, "ideogram", context.Meta.OriginModelName}, "-")] * groupRatio * batchNums), nil

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
func GetName(path string) string {
	return strings.Join([]string{path, strconv.Itoa(int(channeltype.IdeoGram))}, "-")
}
func init() {
	//url-channeltype
	logger.SysLogf("register ideogram channel type start %d", channeltype.IdeoGram)
	registry := rproxy.GetChannelAdaptorRegistry()
	var adaptorBuilder = common.DefaultHttpAdaptorBuilder{
		SetHeaderFunc:    SetHeaderFunc,
		CalcStrategyFunc: CalcStrategyFunc,
		GetUrlFunc:       GetUrlFunc,
	}
	registry.Register(GetName("/ideogram/generate"), adaptorBuilder)
	registry.Register(GetName("/ideogram/edit"), adaptorBuilder)
	registry.Register(GetName("/ideogram/remix"), adaptorBuilder)
	registry.Register(GetName("/ideogram/upscale"), adaptorBuilder)
	registry.Register(GetName("/ideogram/describe"), adaptorBuilder)
	registry.Register(GetName("/ideogram/reframe"), adaptorBuilder)
	logger.SysLogf("register ideogram channel type end %d", channeltype.IdeoGram)

}
