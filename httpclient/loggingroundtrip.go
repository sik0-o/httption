package httpclient

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"

	"go.uber.org/zap"
)

// This type implements the http.RoundTripper interface
type LoggingRoundTripper struct {
	Proxied http.RoundTripper
	logger  *zap.Logger
}

func (lrt LoggingRoundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	if lrt.logger != nil {
		reqOutBytes, err := httputil.DumpRequest(req, true)
		if err != nil {
			return nil, err
		}
		// Do "before sending requests" actions here.
		lrt.logger.Debug("HttpClient Sending request",
			zap.ByteString("request", reqOutBytes),
		)
	}

	// Send the request, get the response (or the error)
	res, err = lrt.Proxied.RoundTrip(req)

	if lrt.logger != nil {
		// Handle the result.
		if err != nil {
			lrt.logger.Error("SolverClient RoundTrip error", zap.Error(err))
		} else {
			respBytes, err := httputil.DumpResponse(res, true)
			if err != nil {
				return nil, err
			}

			lrt.logger.Debug("HttpClient Received response",
				zap.ByteString("response", respBytes),
			)
		}
	}

	return
}

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 7230, section 3.3.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

func inspectContent(r *io.ReadCloser) ([]byte, error) {
	if r == nil {
		return nil, errors.New("ReadCloser pointer is nil")
	}
	if *r == nil {
		return nil, errors.New("ReadCloser *r is nil")
	}

	br, err := io.ReadAll(*r)
	if err != nil {
		return nil, err
	}
	// close prev reader
	(*r).Close()
	//recreate reader
	(*r) = io.NopCloser(bytes.NewBuffer(br))

	return br, nil
}

func inspectHeaders(h http.Header) map[string][]string {
	headers := make(map[string][]string)

	for n, v := range h.Clone() {
		headers[n] = v
	}

	return headers
}
