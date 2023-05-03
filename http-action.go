package httption

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
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

const DISCORD_API_BASE = "https://discord.com/api/v9"

type BaseAction struct {
	beforeAction []HttpAction
	afterAction  []HttpAction

	method string
	url    string

	resp *http.Response

	requestBodyBuffer  []byte
	responseBodyBuffer []byte

	headers map[string]string

	err error

	client *http.Client
}

func NewHttpAction(client *http.Client, method string, url string) HttpAction {
	return NewBaseAction(client, method, url)
}

func NewBaseAction(client *http.Client, method string, url string) *BaseAction {
	return &BaseAction{
		beforeAction: []HttpAction{},
		afterAction:  []HttpAction{},

		method: method,
		url:    url,

		client:  client,
		headers: make(map[string]string),
	}
}

func (ba *BaseAction) setupProxy(proxyURL *url.URL) error {
	switch t := ba.client.Transport.(type) {
	case *http.Transport:
		t.Proxy = http.ProxyURL(proxyURL)
	default:
		return errors.New("BaseAction.setupProxy() cannot set proxy because transport has unknown type")
	}

	return nil
}

func (ha *BaseAction) Do(opts ...Option) error {
	var err error
	var tryN uint

	for {
		tryN++
		retry := false
		isRetry := tryN > 1

		err = ha.do(isRetry, opts...)
		if err != nil {
			// check retry
			if err == ErrTooManyRequests {
				retry = true
			}
		}

		if !retry {
			break
		}
	}

	return err
}

func (ha *BaseAction) do(noSetup bool, opts ...Option) error {
	if !noSetup {
		for _, opt := range opts {
			if err := opt(ha); err != nil {
				return err
			}
		}
	}

	if len(ha.beforeAction) > 0 {
		for _, bact := range ha.beforeAction {
			if err := bact.Do(opts...); err != nil {
				return err
			}
		}
	}

	req, err := prepareRequest(ha.method, ha.url, ha.requestBodyBuffer, ha.headers)
	ha.err = err
	if err != nil {
		return err
	}

	resp, err := ha.client.Do(req)
	ha.err = err
	if err != nil {
		return err
	}
	ha.resp = resp

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = handleResponse(resp, respBody)
	ha.err = err
	if err != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return ErrTooManyRequests
		}

		return err
	}

	ha.responseBodyBuffer = respBody

	if len(ha.afterAction) > 0 {
		for _, aact := range ha.afterAction {
			if err := aact.Do(opts...); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ha *BaseAction) Result(result interface{}) error {
	if ha.responseBodyBuffer == nil {
		return nil
	}

	if ha.resp.Header.Get("content-type") == "application/json" {
		if err := json.Unmarshal(ha.responseBodyBuffer, result); err != nil {
			return err
		}
	}

	return nil
}

func (ha *BaseAction) Error() error {
	return ha.err
}

func (ha *BaseAction) SetHeaders(headers map[string]string) {
	ha.headers = headers
}

func (ha *BaseAction) ResponseBytes() []byte {
	return ha.responseBodyBuffer
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
