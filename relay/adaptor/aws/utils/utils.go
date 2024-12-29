package utils

import (
	"net/http"

	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func WrapErr(err error) *relaymodel.ErrorWithStatusCode {
	return &relaymodel.ErrorWithStatusCode{
		IsChannelResponseError: true,
		StatusCode:             http.StatusInternalServerError,
		Error: relaymodel.Error{
			Message: err.Error(),
		},
	}
}
