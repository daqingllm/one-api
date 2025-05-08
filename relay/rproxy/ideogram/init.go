package ideogram

import (
	"strconv"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/rproxy/common"
)

func init() {
	//url-channeltype
	logger.SysLogf("register ideogram channel type start %d", channeltype.IdeoGram)
	registry := rproxy.GetChannelAdaptorRegistry()
	var adaptorBuilder = common.DefaultHttpAdaptorBuilder{
		SetHeaderFunc:       SetHeaderFunc,
		PreCalcStrategyFunc: PreCalcStrategyFunc,
		GetUrlFunc:          GetUrlFunc,
	}
	var v3AdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		SetHeaderFunc:       SetHeaderFunc,
		PreCalcStrategyFunc: PreV3CalcStrategyFunc,
		GetUrlFunc:          GetUrlFunc,
	}
	registry.Register("/ideogram/generate", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/edit", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/remix", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/upscale", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/describe", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/reframe", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/v1/:model/generate", "POST", strconv.Itoa(int(channeltype.IdeoGram)), v3AdaptorBuilder)
	registry.Register("/ideogram/v1/:model/edit", "POST", strconv.Itoa(int(channeltype.IdeoGram)), v3AdaptorBuilder)
	registry.Register("/ideogram/v1/:model/remix", "POST", strconv.Itoa(int(channeltype.IdeoGram)), v3AdaptorBuilder)
	registry.Register("/ideogram/v1/:model/reframe", "POST", strconv.Itoa(int(channeltype.IdeoGram)), v3AdaptorBuilder)
	registry.Register("/ideogram/v1/:model/replace-background", "POST", strconv.Itoa(int(channeltype.IdeoGram)), v3AdaptorBuilder)
	logger.SysLogf("register ideogram channel type end %d", channeltype.IdeoGram)

}
