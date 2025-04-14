package common

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
	"github.com/songquanpeng/one-api/relay/adaptor"
	"github.com/songquanpeng/one-api/relay/controller"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
	"github.com/songquanpeng/one-api/relay/rproxy"
	"github.com/songquanpeng/one-api/relay/util"
)

type HttpRproxyAdaptor struct {
	Errorhandler      rproxy.ErrorHandler
	ResponseHandler   rproxy.ResponseHandler
	RequestHandler    rproxy.RequestHandler
	BillingCalculator rproxy.BillingCalculator
	channel           *model.Channel
}

func (a *HttpRproxyAdaptor) DoRequest(context *rproxy.RproxyContext) (response rproxy.Response, err *relaymodel.ErrorWithStatusCode) {
	err = a.BillingCalculator.PreCalAndExecute(context)
	if err != nil {
		return nil, err
	}
	newReq, err := a.GetRequestHandler().Handle(context)
	if err != nil {
		go a.BillingCalculator.RollBackPreCalAndExecute(context)
		return nil, err
	}
	resp, error := adaptor.DoRequest(context.SrcContext, newReq.(*http.Request))
	err = a.GetErrorHandler().HandleError(context, resp, error)
	if err != nil {
		go a.BillingCalculator.RollBackPreCalAndExecute(context)
		return nil, err
	}

	e := a.GetResponseHandler().Handle(context, resp)
	if config.DebugUserIds[context.GetUserId()] {
		req := newReq.(*http.Request)
		// 结构化打印请求信息
		logger.DebugForcef(context.SrcContext.Request.Context(),
			"[Request Detail]\nMethod: %s\nURL: %s\nHeaders: %v\n",
			req.Method,
			req.URL.String(),
			req.Header,
		)
		// 结构化打印响应信息
		logger.DebugForcef(context.SrcContext.Request.Context(),
			"[Response Detail]\nStatus: %s\nHeaders: %v\n",
			resp.Status,
			resp.Header,
		)
	}
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
	if httpResp.StatusCode != http.StatusOK {
		return controller.RelayErrorHandler(httpResp)
	}
	return nil
}

type DefaultRequestHandler struct {
	Adaptor               rproxy.RproxyAdaptor
	GetUrlFunc            func(context *rproxy.RproxyContext, channel *model.Channel) (url string, err *relaymodel.ErrorWithStatusCode)
	SetHeaderFunc         func(context *rproxy.RproxyContext, channel *model.Channel, request *http.Request) (err *relaymodel.ErrorWithStatusCode)
	ReplaceBodyParamsFunc func(context *rproxy.RproxyContext, channel *model.Channel, body []byte) (replacedBody []byte, err *relaymodel.ErrorWithStatusCode)
}

func (r *DefaultRequestHandler) Handle(context *rproxy.RproxyContext) (req rproxy.Request, err *relaymodel.ErrorWithStatusCode) {
	originalReq := context.SrcContext.Request
	var fullRequestURL string = ""
	if fullRequestURL, err = r.GetUrlFunc(context, r.Adaptor.GetChannel()); err != nil {
		return nil, err
	}
	contentType := originalReq.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
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
		contentLength := body.Len()
		newReq.Header.Set("Content-Length", fmt.Sprintf("%d", contentLength))
		for k, v := range originalReq.Header {
			if k != "Content-Type" {
				newReq.Header.Set(k, v[0])
			}
		}
		if r.SetHeaderFunc != nil {
			r.SetHeaderFunc(context, r.Adaptor.GetChannel(), newReq)
		}
		newReq.Header.Set("Content-Type", writer.FormDataContentType())
		newReq.Header.Del("Accept-Encoding")
		return newReq, nil
	}

	requestBody, e := io.ReadAll(context.SrcContext.Request.Body)
	if e != nil {
		return nil, relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "read_request_body_failed", "read_request_body_failed")
	}
	defer context.SrcContext.Request.Body.Close() // 确保关闭 Body
	context.SrcContext.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

	if r.ReplaceBodyParamsFunc != nil {
		newRequestBody, _ := r.ReplaceBodyParamsFunc(context, r.Adaptor.GetChannel(), requestBody)
		if newRequestBody != nil {
			requestBody = newRequestBody
		}
	}
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
	newReq.Header.Del("Accept-Encoding")
	return newReq, nil
}

type DefaultResponseHandler struct {
}

func (r *DefaultResponseHandler) Handle(context *rproxy.RproxyContext, resp rproxy.Response) (err *relaymodel.ErrorWithStatusCode) {
	httpResp, ok := resp.(*http.Response)
	if !ok {
		return relaymodel.NewErrorWithStatusCode(http.StatusInternalServerError, "invalid_response", "invalid_response")
	}
	var result any
	if context.Meta.IsStream {
		result, err = util.StreamResponseHandle(context.SrcContext, httpResp)

	} else {
		result, err = util.ResponseHandle(context.SrcContext, httpResp)
	}
	context.ResolvedResponse = result
	return
}
