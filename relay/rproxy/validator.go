package rproxy

import "github.com/songquanpeng/one-api/relay/model"

type Validator interface {
	Validate() *model.ErrorWithStatusCode
}
