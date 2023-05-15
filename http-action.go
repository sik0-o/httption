package httption

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HttpAction is an interface to perform http requests and handle its responses.
// It has beforeAction and afterAction callbacks that it is also HttpActions.
type HttpAction interface {
	Do(opts ...Option) error         // perform http action
	Result(result interface{}) error // unmarshal result to struct
	Error() error                    // return last action error
}

var (
	ErrInvalidPayment      = errors.New("Invalid payment")
	ErrNeedEmailAuthorize  = errors.New("This client needs to be authorized for purchases. We've sent you an email. Click the link on the email and then retry the purchase.")
	ErrEmptyPaymentMethods = errors.New("Empty payment methods")
	ErrTooManyRequests     = errors.New("Too many requests")
)

var ErrEmptyRequest = errors.New("action request is empty. Do Setup() method before action request performed")

type BaseAction struct {
	name string
	// Callbacks
	// beforeAction []HttpAction
	// afterAction  []HttpAction

	// HTTP-params
	client             *http.Client
	method             string
	url                string
	request            *http.Request
	response           *http.Response
	requestBodyBuffer  []byte
	responseBodyBuffer []byte
	headers            map[string]string

	logger *zap.Logger

	// Response Handlers
	statusCodeHandlers map[int]StatusCodeHandlerFunc

	// Retry
	tryCount   uint
	maxRetry   uint
	needRetry  bool
	retryDelay time.Duration

	// Misc
	err error
}

func NewHttpAction(client *http.Client, method string, url string) HttpAction {
	return NewBaseAction(client, method, url)
}

func NewBaseAction(client *http.Client, method string, url string) *BaseAction {
	l, _ := zap.NewDevelopment()

	return &BaseAction{
		// beforeAction: []HttpAction{},
		// afterAction:  []HttpAction{},

		method: method,
		url:    url,

		client:  client,
		headers: make(map[string]string),

		logger: l,
	}
}

func (ba *BaseAction) Name() string {
	return ba.name
}

type ProxiedTransport interface {
	SetProxy(proxyURL *url.URL) error
}

func (ba *BaseAction) SetupProxyURL(proxyURL *url.URL) error {
	switch t := ba.client.Transport.(type) {
	case *http.Transport:
		t.Proxy = http.ProxyURL(proxyURL)
	default:
		return errors.New("BaseAction.SetupProxy() cannot set proxy because transport has unknown type")
	}

	return nil
}

func (ba *BaseAction) SetLogger(logger *zap.Logger) {
	ba.logger = logger
}

func (ba *BaseAction) Logger() *zap.Logger {
	return ba.logger
}

func (ha *BaseAction) SetHeaders(headers map[string]string) {
	ha.headers = headers
}

// Setup do an option setup than prepare action http-request if it not build previously
func (ba *BaseAction) Setup(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(ba); err != nil {
			return err
		}
	}

	if ba.request == nil {
		req, err := prepareRequest(ba.method, ba.url, ba.requestBodyBuffer, ba.headers)
		ba.err = err
		if err != nil {
			return err
		}

		ba.request = req
	}

	return nil
}

// Repeat method repeats the http request.
func (ba *BaseAction) Repeat(setupAction bool, opts ...Option) error {
	ba.log(zap.DebugLevel, "action Repeat() Setup")
	if err := ba.Setup(opts...); err != nil {
		ba.log(zap.ErrorLevel, "action Repeat() Setup", zap.Error(err))
		return err
	}
	ba.log(zap.DebugLevel, "action Repeat() do")

	return ba.do()
}

func (ba *BaseAction) Do(opts ...Option) error {
	ba.log(zap.DebugLevel, "action Do() Setup")
	if err := ba.Setup(opts...); err != nil {
		ba.log(zap.ErrorLevel, "action Do() Setup", zap.Error(err))
		return err
	}

	var retry bool

	// Next just make a do
	for {
		if ba.maxRetry > 0 {
			retry = ba.tryCount <= ba.maxRetry
		} else {
			retry = false
		}
		err := ba.do()

		if err == nil {
			retry = false
			// no errors -> quit cycle.
			break
		} else {
			// when error and we have no retry -> throw error
			ba.log(zap.ErrorLevel, "action Do() do", zap.Error(err))
			// exit when no retry
			if !retry {
				return err
			}
		}
		// else process again
		if ba.retryDelay > 0 {
			ba.log(zap.DebugLevel, "waiting before retry", zap.Duration("delay", ba.retryDelay))
			<-time.After(ba.retryDelay)
		}
	}

	// repeat an action if retry is need
	// it happpend when success action need a repeat
	if ba.needRetry {
		return ba.Repeat(false)
	}

	return nil
}

func (ha *BaseAction) Result(result interface{}) error {
	if ha.responseBodyBuffer == nil {
		return nil
	}

	if ha.response.Header.Get("content-type") == "application/json" {
		if err := json.Unmarshal(ha.responseBodyBuffer, result); err != nil {
			return err
		}
	}

	return nil
}

func (ha *BaseAction) Error() error {
	return ha.err
}

func (ha *BaseAction) ResponseBytes() []byte {
	return ha.responseBodyBuffer
}

func (ba *BaseAction) do() error {

	ba.tryCount++
	// if len(ha.beforeAction) > 0 {
	// 	for _, bact := range ha.beforeAction {
	// 		if err := bact.Do(opts...); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	ba.log(zap.DebugLevel, "do action", zap.Uint("attempt", ba.tryCount))

	if ba.request == nil {
		return ErrEmptyRequest
	}

	ba.log(zap.DebugLevel, "sending request")
	resp, err := ba.client.Do(ba.request)
	ba.err = err
	if err != nil {
		return err
	}
	ba.log(zap.DebugLevel, "response received")
	ba.response = resp

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	ba.log(zap.DebugLevel, "response body read")

	if ba.statusCodeHandlers != nil {
		if statusHandler, ok := ba.statusCodeHandlers[resp.StatusCode]; ok {
			if statusHandler(ba.client) {
				return errors.New("resp.statusHandler")
			}
		}
	}

	err = handleResponse(resp, respBody)
	ba.err = err
	if err != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return ErrTooManyRequests
		}

		return err
	}

	ba.responseBodyBuffer = respBody

	ba.log(zap.DebugLevel, "response handled")

	// if len(ha.afterAction) > 0 {
	// 	for _, aact := range ha.afterAction {
	// 		if err := aact.Do(opts...); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	return nil
}

func prepareRequest(method string, url string, bodyBytes []byte, headers map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	return req, nil
}

func handleResponse(resp *http.Response, respBody []byte) error {
	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrTooManyRequests
	}

	if resp.StatusCode == http.StatusBadRequest {
		if respBody != nil {
			if resp.Header.Get("content-type") == "application/json" {
				respData := map[string]any{}
				if err := json.Unmarshal(respBody, &respData); err != nil {
					return err
				}

				code, _ := respData["code"].(int)
				switch code {
				case 100008:
					return ErrInvalidPayment
				case 100056:
					return ErrNeedEmailAuthorize
				}

				if msg, ok := respData["message"].(string); ok {
					if msg == "Invalid payment" {
						return ErrInvalidPayment
					}
				}
			}

			return errors.New("BadRequest")
		}

		return errors.New("BadRequest noBody")
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return errors.New(resp.Request.URL.String() + " status is not ok: " + string(respBody))
	}

	return nil
}

func (ba *BaseAction) log(lvl zapcore.Level, msg string, fields ...zap.Field) {
	if ba.logger == nil {
		return
	}

	fields = append([]zap.Field{zap.String("action_name", ba.name)}, fields...)

	ba.logger.Log(lvl, msg, fields...)
}
