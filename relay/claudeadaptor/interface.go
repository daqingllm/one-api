package claude_adaptor

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/relay/adaptor/anthropic"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/model"
)

type Adaptor interface {
	DoRequest(c *gin.Context, request *anthropic.Request, meta *meta.Meta) (usage *anthropic.Usage, err *model.ErrorWithStatusCode)
}
