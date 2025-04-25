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
	registry.Register("/ideogram/generate", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/edit", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/remix", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/upscale", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/describe", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	registry.Register("/ideogram/reframe", "POST", strconv.Itoa(int(channeltype.IdeoGram)), adaptorBuilder)
	logger.SysLogf("register ideogram channel type end %d", channeltype.IdeoGram)

}
