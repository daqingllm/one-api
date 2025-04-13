package rproxy

import (
	"strings"
	"sync"
)

var (
	instance = &ChannelAdaptorRegistry{
		adaptorBuilders: make(map[string]AdaptorBuilder),
	}
	once sync.Once
)

type ChannelAdaptorRegistry struct {
	adaptorBuilders map[string]AdaptorBuilder
}

type AdaptorBuilder interface {
	Build() (adaptor RproxyAdaptor)
}

func (r *ChannelAdaptorRegistry) Register(name string, builder AdaptorBuilder) {
	name = strings.ToLower(name)
	if r.adaptorBuilders[name] != nil {
		return
	}
	r.adaptorBuilders[name] = builder
}

func (r *ChannelAdaptorRegistry) GetAdaptor(name string) RproxyAdaptor {
	name = strings.ToLower(name)
	if r.adaptorBuilders[name] == nil {
		return nil
	}
	return r.adaptorBuilders[name].Build()
}
func GetChannelAdaptorRegistry() *ChannelAdaptorRegistry {
	once.Do(func() {
		instance = &ChannelAdaptorRegistry{
			adaptorBuilders: make(map[string]AdaptorBuilder),
		}
	})
	return instance
}
