package httption

import (
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
		return ba.setupProxy(proxyURL)
	}
}
