package rproxy

import (
	"strings"
	"sync"
)

var (
	instance = &ChannelAdaptorRegistry{
		adaptorBuilders: make([]RouterAdaptor, 0),
	}
	once sync.Once
)

// 修改注册表结构
type RoutePattern struct {
	Method      string
	PathPattern string // 包含参数的模式，如 "/v1beta/models/:model/*action"
	ChannelType string
}

type RouterAdaptor struct {
	pattern RoutePattern
	builder AdaptorBuilder
}
type ChannelAdaptorRegistry struct {
	adaptorBuilders []RouterAdaptor
}

type AdaptorBuilder interface {
	Build() (adaptor RproxyAdaptor)
}

// RegisterBatch 批量注册相同的构建器到多个路由模式
// 将pathPattern, method, channelType作为一个整体进行批量注册
func (r *ChannelAdaptorRegistry) RegisterBatch(patterns []RoutePattern, builder AdaptorBuilder) {
	for _, pattern := range patterns {
		r.Register(pattern.PathPattern, pattern.Method, pattern.ChannelType, builder)
	}
}

// RegisterMultiBuilders 为同一个路由模式注册多个构建器
// 将pathPattern, method, channelType作为一个整体，注册多个构建器
func (r *ChannelAdaptorRegistry) RegisterMultiBuilders(pattern RoutePattern, builders []AdaptorBuilder) {
	for _, builder := range builders {
		r.Register(pattern.PathPattern, pattern.Method, pattern.ChannelType, builder)
	}
}

// RegisterForChannelTypes 为多个渠道类型注册相同的路由模式和构建器
// 将同一个路由模式和方法注册到多个渠道类型
func (r *ChannelAdaptorRegistry) RegisterForChannelTypes(pathPattern, method string, channelTypes []string, builder AdaptorBuilder) {
	for _, channelType := range channelTypes {
		r.Register(pathPattern, method, channelType, builder)
	}
}

func (r *ChannelAdaptorRegistry) Register(pathPattern, method, channelType string, builder AdaptorBuilder) {
	r.adaptorBuilders = append(r.adaptorBuilders, RouterAdaptor{
		pattern: RoutePattern{
			Method:      strings.ToUpper(method),
			PathPattern: pathPattern,
			ChannelType: strings.ToLower(channelType),
		},
		builder: builder,
	})
}

func (r *ChannelAdaptorRegistry) GetAdaptor(path, method, channelType string) RproxyAdaptor {
	method = strings.ToUpper(method)
	channelType = strings.ToLower(channelType)
	pathSegments := splitPath(path)

	for _, entry := range r.adaptorBuilders {
		if entry.pattern.Method != method || entry.pattern.ChannelType != channelType {
			continue
		}

		if matchPattern(entry.pattern.PathPattern, pathSegments) {
			return entry.builder.Build()
		}
	}
	return nil
}
func GetChannelAdaptorRegistry() *ChannelAdaptorRegistry {
	once.Do(func() {
		instance = &ChannelAdaptorRegistry{
			adaptorBuilders: make([]RouterAdaptor, 0),
		}
	})
	return instance
}

// 路径分割和匹配工具函数
func splitPath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

func matchPattern(pattern string, requestSegments []string) bool {
	patternSegments := splitPath(pattern)
	if len(patternSegments) != len(requestSegments) && !hasWildcard(patternSegments) {
		return false
	}

	for i, pSeg := range patternSegments {
		if i >= len(requestSegments) {
			return false
		}

		// 处理转义冒号 \:
		if strings.Contains(pSeg, "\\:") {
			// 去掉转义符号，只比较实际的冒号和后面的内容
			pSegParts := strings.SplitN(pSeg, "\\:", 2)
			reqSegParts := strings.SplitN(requestSegments[i], ":", 2)

			if len(reqSegParts) != 2 || reqSegParts[1] != pSegParts[1] {
				return false
			}
			continue
		}

		// 处理通配符 *
		if pSeg == "*" || (strings.HasPrefix(pSeg, "*") && i == len(patternSegments)-1) {
			return true
		}

		// 处理命名参数 :
		if strings.HasPrefix(pSeg, ":") {
			continue
		}

		// 精确匹配
		if pSeg != requestSegments[i] {
			return false
		}
	}
	return len(patternSegments) == len(requestSegments)
}

func hasWildcard(segments []string) bool {
	for _, seg := range segments {
		if seg == "*" || strings.HasPrefix(seg, "*") {
			return true
		}
	}
	return false
}
