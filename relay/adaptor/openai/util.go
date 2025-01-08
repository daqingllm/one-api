package openai

import (
	"context"
	"fmt"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/model"
)

func ErrorWrapper(err error, code string, statusCode int) *model.ErrorWithStatusCode {
	logger.Error(context.TODO(), fmt.Sprintf("[%s]%+v", code, err))

	Error := model.Error{
		Message: err.Error(),
		Type:    "Aihubmix_api_error",
		Code:    code,
	}
	return &model.ErrorWithStatusCode{
		Error:      Error,
		StatusCode: statusCode,
	}
}

func ChannelErrorWrapper(err error, code string, statusCode int) *model.ErrorWithStatusCode {
	Error := model.Error{
		Message: err.Error(),
		Type:    "Aihubmix_api_error",
		Code:    code,
	}
	return &model.ErrorWithStatusCode{
		IsChannelResponseError: true,
		Error:                  Error,
		StatusCode:             statusCode,
	}
}
