package common

import (
	"github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type QuotaValidator struct {
	ctx *rproxy.RproxyContext
}

func (qv *QuotaValidator) Validate() *model.ErrorWithStatusCode {
	return nil
}
