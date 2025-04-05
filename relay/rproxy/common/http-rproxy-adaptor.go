package common

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/controller"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
)

type HttpRproxyAdaptor struct {
	Errorhandler      rproxy.ErrorHandler
	ResponseHandler   rproxy.ResponseHandler
	RequestHandler    rproxy.RequestHandler
	BillingCalculator rproxy.BillingCalculator
	channel           *model.Channel
}

func (a *HttpRproxyAdaptor) DoRequest(context *rproxy.RproxyContext) (response rproxy.Response, err *relaymodel.ErrorWithStatusCode) {
	a.BillingCalculator.PreCalAndExecute(context)
	newReq, err := a.GetRequestHandler().Handle(context)
	if err != nil {
		return nil, err
	}
	logger.Infof(context.SrcContext, "Request : %v", newReq)
	resp, error := adaptor.DoRequest(context.SrcContext, newReq.(*http.Request))
	err = a.GetErrorHandler().HandleError(context, resp, error)
	if err != nil {
		go a.BillingCalculator.RollBackPreCalAndExecute(context)
		return nil, err
	}
	e := a.GetResponseHandler().Handle(context, resp)
	if e != nil {
		return nil, e
	}
	go a.BillingCalculator.PostCalcAndExecute(context)
	return nil, nil
}

func (a *HttpRproxyAdaptor) SetChannel(channel *model.Channel) {
	a.channel = channel
}

func (a *HttpRproxyAdaptor) GetChannel() *model.Channel {
	return a.channel
}
func (a *HttpRproxyAdaptor) GetRequestHandler() rproxy.RequestHandler {
	return a.RequestHandler
}

func (a *HttpRproxyAdaptor) GetResponseHandler() rproxy.ResponseHandler {
	return a.ResponseHandler
}

func (a *HttpRproxyAdaptor) GetErrorHandler() rproxy.ErrorHandler {
	return a.Errorhandler
}

type DefaultErrorHandler struct {
}

func (r *DefaultErrorHandler) HandleError(context *rproxy.RproxyContext, resp rproxy.Response, e error) (err *relaymodel.ErrorWithStatusCode) {
	if e != nil {
		logger.Errorf(context.SrcContext, "DoRequest failed: %s", e.Error())
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "do_request_failed", fmt.Errorf("do request failed: %s", adaptor.MaskBaseURL(e.Error(), "")).Error())
	}
	httpResp, ok := resp.(*http.Response)
	if !ok {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "invalid_response", "invalid_response")
	}
	logger.SysLogf("http response %v", httpResp)
	if httpResp.StatusCode != http.StatusOK {
		return controller.RelayErrorHandler(httpResp)
	}
	return nil
}

type DefaultRequestHandler struct {
	Adaptor       rproxy.RproxyAdaptor
	GetUrlFunc    func(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode)
	SetHeaderFunc func(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode)
}

func (r *DefaultRequestHandler) Handle(context *rproxy.RproxyContext) (req rproxy.Request, err *relaymodel.ErrorWithStatusCode) {
	originalReq := context.SrcContext.Request
	var fullRequestURL string = ""
	if fullRequestURL, err = r.GetUrlFunc(context, r.Adaptor.GetChannel()); err != nil {
		return nil, err
	}
	contentType := originalReq.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// // 提取原始boundary
		if originalReq.MultipartForm == nil {
			e := originalReq.ParseMultipartForm(32 << 20)
			if e != nil {
				return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "parse_multipart_failed", "parse_multipart_failed")
			}
		}
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for key, files := range originalReq.MultipartForm.File {
			for _, fileHeader := range files {
				file, e := fileHeader.Open()
				if e != nil {
					return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "open_file_failed", "open_file_failed")
				}
				defer file.Close()

				part, e := writer.CreateFormFile(key, fileHeader.Filename)
				if e != nil {
					writer.Close()
					return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "create_form_file_failed", "create_form_file_failed")
				}

				_, e = io.Copy(part, file)
				if e != nil {
					writer.Close()
					return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "copy_file_failed", "copy_file_failed")
				}
			}
		}

		// 处理文本部分
		for key, values := range originalReq.MultipartForm.Value {
			for _, value := range values {
				if err := writer.WriteField(key, value); err != nil {
					writer.Close()
					return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "write_field_failed", "failed to write form field")
				}
			}
		}
		writer.Close()
		newReq, e := http.NewRequest(originalReq.Method, fullRequestURL, body)
		if e != nil {
			return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "new_request_failed", "new_request_failed")
		}
		// 显式设置 Content-Length
		contentLength := body.Len()
		newReq.Header.Set("Content-Length", fmt.Sprintf("%d", contentLength))
		// 打印新请求信息
		logger.Infof(context.SrcContext, "New Request Method: %s", newReq.Method)
		logger.Infof(context.SrcContext, "New Request URL: %s", newReq.URL.String())
		for k, v := range originalReq.Header {
			if k != "Content-Type" {
				newReq.Header.Set(k, v[0])
			}
		}
		if r.SetHeaderFunc != nil {
			r.SetHeaderFunc(context, r.Adaptor.GetChannel(), newReq)
		}
		newReq.Header.Set("Content-Type", writer.FormDataContentType())
		return newReq, nil
	}

	requestBody, e := io.ReadAll(context.SrcContext.Request.Body)
	if e != nil {
		return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "read_request_body_failed", "read_request_body_failed")
	}
	defer context.SrcContext.Request.Body.Close() // 确保关闭 Body

	newReq, e := http.NewRequest(originalReq.Method, fullRequestURL, bytes.NewBuffer(requestBody))
	//copy header
	for k, v := range originalReq.Header {
		newReq.Header.Set(k, v[0])
	}
	if e != nil {
		return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "new_request_failed", "new_request_failed")
	}
	if r.SetHeaderFunc != nil {
		r.SetHeaderFunc(context, r.Adaptor.GetChannel(), newReq)
	}
	return newReq, nil
}

type DefaultResponseHandler struct {
}

func (r *DefaultResponseHandler) Handle(context *rproxy.RproxyContext, resp rproxy.Response) (err *relaymodel.ErrorWithStatusCode) {
	httpResp, ok := resp.(*http.Response)
	if !ok {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "invalid_response", "invalid_response")
	}
	responseBody, e := io.ReadAll(httpResp.Body)
	if e != nil {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "read_response_body_failed", "read_response_body_failed")
	}
	e = httpResp.Body.Close()
	if e != nil {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "close_response_body_failed", "close_response_body_failed")
	}

	httpResp.Body = io.NopCloser(bytes.NewBuffer(responseBody))
	for k, v := range httpResp.Header {
		context.SrcContext.Writer.Header().Set(k, v[0])
	}
	context.SrcContext.Writer.WriteHeader(httpResp.StatusCode)
	_, e = io.Copy(context.SrcContext.Writer, httpResp.Body)
	if e != nil {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "copy_response_body_failed", "copy_response_body_failed")
	}
	e = httpResp.Body.Close()
	if e != nil {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "close_response_body_failed", "close_response_body_failed")
	}
	return nil
}
