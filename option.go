package httption

import (
	"net/http"
	"net/url"
)

type Option func(ba *BaseAction) error

func WithHeaders(headers map[string]string) Option {
	return func(ba *BaseAction) error {
		ba.headers = headers

		return nil
	}
}

func WithProxyURL(proxyURL *url.URL) Option {
	return func(ba *BaseAction) error {
		return ba.SetupProxyURL(proxyURL)
	}
}

type StatusCodeHandlerFunc func(client *http.Client) bool

func WithStatusCodeHandler(code int, h StatusCodeHandlerFunc) Option {
	return func(ba *BaseAction) error {
		ba.statusCodeHandlers[code] = h

		return nil
	}
}
