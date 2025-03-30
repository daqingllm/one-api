package rproxy

import (
	"github.com/gin-gonic/gin"
)

type WeaverFactory interface {
	GetWeaver(ctx *gin.Context) (weaver Weaver)
}

