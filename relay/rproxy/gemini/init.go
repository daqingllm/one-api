package gemini

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
		PreCalcStrategyFunc:  PreCalcStrategyFunc,
		PostCalcStrategyFunc: PostCalcStrategyFunc,
		GetUrlFunc:           GetUrlFunc,
		StreamHandFunc:       StreamHandFunc,
	}

	var vertexAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		PreCalcStrategyFunc:  PreCalcStrategyFunc,
		PostCalcStrategyFunc: PostCalcStrategyFunc,
		GetUrlFunc:           GetVertexUrlFunc,
		SetHeaderFunc:        SetVertexHeaderFunc,
		StreamHandFunc:       StreamHandFunc,
	}

	var fileAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		GetBillingCalculator: func() rproxy.BillingCalculator {
			return &common.NOPBillingCalculator{}
		},
		GetUrlFunc:    GetFileUrlFunc,
		SetHeaderFunc: SetFileHeaderFunc,
	}

	// var cacheAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
	// 	PostCalcStrategyFunc: CachePostCalcStrategyFunc,
	// 	GetUrlFunc:           GetUrlFunc,
	// 	SetHeaderFunc:        SetFileHeaderFunc,
	// }

	logger.SysLogf("register gemin response channel type start %d", channeltype.Gemini)

	// 使用批量注册方式注册Gemini模型接口
	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:modelAction", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, adaptorBuilder)

	// 注册Gemin转VertexAI模型接口
	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:modelAction", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.VertextAI))},
	}, vertexAdaptorBuilder)

	// 批量注册Gemini文件相关接口
	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/files", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
		{PathPattern: "/gemini/v1beta/files/:filename", Method: "GET", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
		{PathPattern: "/gemini/v1beta/files", Method: "GET", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
		{PathPattern: "/gemini/v1beta/files/:filename", Method: "DELETE", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, fileAdaptorBuilder)

	// 批量注册Gemini缓存内容相关接口
	// registry.RegisterBatch([]rproxy.RoutePattern{
	// 	{PathPattern: "/v1beta/cachedContents/:cachedname", Method: "GET", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	// 	{PathPattern: "/v1beta/cachedContents/:cachedname", Method: "PATCH", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	// 	{PathPattern: "/v1beta/cachedContents/:cachedname", Method: "DELETE", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	// 	{PathPattern: "/v1beta/cachedContents", Method: "GET", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	// 	{PathPattern: "/v1beta/cachedContents", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	// }, cacheAdaptorBuilder)

	//原生vertex ai支持
	registry.Register("/gemini/v1/projects/:VertexAIProjectID/locations/:region/publishers/google/models/:modelAction",
		"POST", strconv.Itoa(int(channeltype.VertextAI)), vertexAdaptorBuilder)

	logger.SysLogf("register gemin response channel type end %d", channeltype.Gemini)

}
