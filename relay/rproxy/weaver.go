package rproxy

import "github.com/songquanpeng/one-api/relay/model"

type Weaver interface {
	Weave() (err *model.ErrorWithStatusCode)
	GetAuthChecker() AuthChecker
	GetFaultTolerancer() FaultTolerancer
	GetChannelSelector() ChannelSelector
}

type DefaultWeaver struct {
	AuthChecker     AuthChecker
	RproxyAdaptor   RproxyAdaptor
	FaultTolerancer FaultTolerancer
}

func (w *DefaultWeaver) Weave() (err *model.ErrorWithStatusCode) {

	if w.AuthChecker != nil {
		if checkResult, err := w.AuthChecker.check(); err != nil {
			return err
		} else if !checkResult {
			return &model.ErrorWithStatusCode{
				Error: model.Error{
					Code:    "auth_failed",
					Message: "auth failed",
				},
			}
		}
	}
	return nil
	// w.FaultTolerancer.FaultTolerance(w.RproxyContext)
}

func (w *DefaultWeaver) GetAuthChecker() AuthChecker {
	return w.AuthChecker
}

func (w *DefaultWeaver) GetFaultTolerancer() FaultTolerancer {
	return w.FaultTolerancer
}

func (w *DefaultWeaver) GetRproxyAdaptor() RproxyAdaptor {
	return w.RproxyAdaptor
}

func (w *DefaultWeaver) GetChannelSelector() ChannelSelector {
	panic("implement me")
}
