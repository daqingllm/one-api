package rproxy

import "github.com/songquanpeng/one-api/relay/model"

type AuthChecker interface {
	Check() (result bool, err *model.ErrorWithStatusCode)
}
