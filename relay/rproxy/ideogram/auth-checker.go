package ideogram

import "github.com/songquanpeng/one-api/relay/model"

type IdeoGramAuthChecker struct {
}

func (a *IdeoGramAuthChecker) check() (checkResult bool, err *model.ErrorWithStatusCode) {
	return true, nil
}
