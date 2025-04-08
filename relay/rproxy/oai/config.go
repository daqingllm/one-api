package oai

import (
	"fmt"
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
	"reframe-ideogram":             0.01 * ratio.USD,
	"upscale-ideogram-UPSCALE":     0.06 * ratio.USD,
	"describe-ideogram-DESCRIBE":   0.01 * ratio.USD,
}

func SetHeaderFunc(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode) {
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))
	return nil
}

func CalcStrategyFunc(context *rproxy.RproxyContext, channel *model.Channel, groupRatio float64) (preConsumedQuota int64, err *relaymodel.ErrorWithStatusCode) {
	path := context.SrcContext.Request.URL.Path
	parts := strings.FieldsFunc(path, func(c rune) bool { return c == '/' })
	if len(parts) == 0 {
		return 0, &relaymodel.ErrorWithStatusCode{
			StatusCode: http.StatusBadRequest,
			Error: relaymodel.Error{
				Message: "invalid_path",
				Code:    "INVALID_PATH",
			},
		}
	}
	lastSegment := parts[len(parts)-1]

	return int64(abilityChannelModelPrices[strings.Join([]string{lastSegment, "ideogram", context.Meta.OriginModelName}, "-")] * groupRatio * 1000), nil

}
func GetName(path string) string {
	return strings.Join([]string{path, strconv.Itoa(int(channeltype.IdeoGram))}, "-")
}
func init() {
	//url-channeltype
	logger.SysLogf("register openai response channel type start %d", channeltype.IdeoGram)
	registry := rproxy.GetChannelAdaptorRegistry()
	var adaptorBuilder = common.DefaultHttpAdaptorBuilder{
		SetHeaderFunc:    SetHeaderFunc,
		CalcStrategyFunc: CalcStrategyFunc,
	}
	registry.Register(GetName("/v1/responses"), adaptorBuilder)
	logger.SysLogf("register openai response channel type end %d", channeltype.IdeoGram)

}
