package util

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/common/render"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/model"
)

func ResponseHandler(c *gin.Context, resp *http.Response) (responseBody []byte, err *model.ErrorWithStatusCode) {
	responseBody, e := io.ReadAll(resp.Body)
	if e != nil {
		return nil, openai.ErrorWrapper(e, "read_response_body_failed", http.StatusInternalServerError)
	}
	e = resp.Body.Close()

	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	_, e = io.Copy(c.Writer, resp.Body)
	if e != nil {
		return responseBody, openai.ErrorWrapper(e, "copy_response_body_failed", http.StatusRequestTimeout)
	}
	e = resp.Body.Close()
	if e != nil {
		return responseBody, openai.ErrorWrapper(e, "close_response_body_failed", http.StatusInternalServerError)
	}
	return responseBody, nil
}

func StreamResponseHandler(c *gin.Context, resp *http.Response) (responseBody []byte, e *model.ErrorWithStatusCode) {
	ctx := c.Request.Context()
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := strings.Index(string(data), "\n"); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	common.SetEventStreamHeaders(c)
	for scanner.Scan() {
		data := scanner.Text()
		render.RawData(c, data)
		if len(data) < 6 || !strings.HasPrefix(data, "data:") {
			continue
		}
		data = strings.TrimPrefix(data, "data:")
		data = strings.TrimSpace(data)

	}
	if err := scanner.Err(); err != nil {
		logger.Error(ctx, "error reading stream: "+err.Error())
		return nil, openai.ErrorWrapper(err, "read_stream_failed", http.StatusInternalServerError)
	}
	err := resp.Body.Close()
	if err != nil {
		return nil, openai.ErrorWrapper(err, "close_response_body_failed", http.StatusInternalServerError)
	}
	return nil, nil
}
