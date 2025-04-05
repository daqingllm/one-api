package rproxy

type WeaverBuilder interface {
	GetWeaverBuilder() *WeaverBuilder
	Build() (weaver Weaver)
}
