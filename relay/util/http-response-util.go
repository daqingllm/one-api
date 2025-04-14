package util

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/render"
	"github.com/songquanpeng/one-api/relay/adaptor/openai"
	"github.com/songquanpeng/one-api/relay/model"
)

func ResponseHandle(c *gin.Context, resp *http.Response) (result any, e *model.ErrorWithStatusCode) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	defer resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	for k, v := range resp.Header {
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "copy_response_body_failed", http.StatusRequestTimeout)
	}
	return responseBody, nil
}

func StreamResponseHandle(c *gin.Context, resp *http.Response) (result any, e *model.ErrorWithStatusCode) {
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
	var completedResponse string
	for scanner.Scan() {
		data := scanner.Text()
		render.RawData(c, data)
		if len(data) < 6 || !strings.HasPrefix(data, "data: {\"type\":\"response.completed\"") {
			continue
		}
		data = strings.TrimPrefix(data, "data:")
		data = strings.TrimSpace(data)
		completedResponse = data
	}
	return []byte(completedResponse), nil
}
