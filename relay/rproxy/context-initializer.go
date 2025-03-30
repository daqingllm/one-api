package rproxy

type ContextInitializer interface {
	Initialize(context *RproxyContext)
}
