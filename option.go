package httption

import (
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
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

func WithLogger(logger *zap.Logger) Option {
	return func(ba *BaseAction) error {
		ba.logger = logger
		return nil
	}
}

func WithRetry(maxRetry *uint, needRetry *bool, retryDelay *time.Duration) Option {
	return func(ba *BaseAction) error {
		if maxRetry != nil {
			ba.maxRetry = *maxRetry
		}
		if needRetry != nil {
			ba.needRetry = *needRetry
		}

		if retryDelay != nil {
			ba.retryDelay = *retryDelay
		}

		return nil
	}
}

func WithMaxRetry(maxRetry uint) Option {
	return func(ba *BaseAction) error {
		ba.maxRetry = maxRetry
		return nil
	}
}

func WithNeedRetry(needRetry bool) Option {
	return func(ba *BaseAction) error {
		ba.needRetry = needRetry
		return nil
	}
}

func WithRetryDelay(retryDelay time.Duration) Option {
	return func(ba *BaseAction) error {
		ba.retryDelay = retryDelay
		return nil
	}
}

type StatusCodeHandlerFunc func(client *http.Client) bool

func WithStatusCodeHandler(code int, h StatusCodeHandlerFunc) Option {
	return func(ba *BaseAction) error {
		ba.statusCodeHandlers[code] = h

		return nil
	}
}
