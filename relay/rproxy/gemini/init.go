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

	var imageAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		GetUrlFunc:          GetUrlFunc,
		PreCalcStrategyFunc: ImagePreCalcStrategyFunc,
	}

	var videoAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		GetUrlFunc:          GetUrlFunc,
		PreCalcStrategyFunc: VideoPreCalcStrategyFunc,
	}

	var vertexImageAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		GetUrlFunc:          GetVertexUrlFunc,
		PreCalcStrategyFunc: ImagePreCalcStrategyFunc,
		SetHeaderFunc:       SetVertexHeaderFunc,
	}

	var vertexVideoAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
		GetUrlFunc:          GetVertexUrlFunc,
		PreCalcStrategyFunc: VideoPreCalcStrategyFunc,
		SetHeaderFunc:       SetVertexHeaderFunc,
	}

	// var cacheAdaptorBuilder = common.DefaultHttpAdaptorBuilder{
	// 	PostCalcStrategyFunc: CachePostCalcStrategyFunc,
	// 	GetUrlFunc:           GetUrlFunc,
	// 	SetHeaderFunc:        SetFileHeaderFunc,
	// }

	logger.SysLogf("register gemin response channel type start %d", channeltype.Gemini)

	// 使用批量注册方式注册Gemini模型接口
	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:generateContent", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, adaptorBuilder)

	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:streamGenerateContent", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, adaptorBuilder)

	// 使用批量注册方式注册Gemini模型接口
	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:generateContent", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, vertexAdaptorBuilder)

	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:streamGenerateContent", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, vertexAdaptorBuilder)

	// 使用批量注册方式注册Gemini模型接口
	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:predict", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, imageAdaptorBuilder)

	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:predictLongRunning", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
	}, videoAdaptorBuilder)

	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:predict", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.VertextAI))},
	}, vertexImageAdaptorBuilder)

	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/v1beta/models/:model\\:predictLongRunning", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.VertextAI))},
	}, vertexVideoAdaptorBuilder)

	// 批量注册Gemini文件相关接口
	registry.RegisterBatch([]rproxy.RoutePattern{
		{PathPattern: "/gemini/upload/v1beta/files", Method: "POST", ChannelType: strconv.Itoa(int(channeltype.Gemini))},
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
