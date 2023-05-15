package httpclient

// HttpClient - может потребоваться:
// 1. логирование запросов
// 2. Делать POST и GET запросы.
// 3. API ключ может передаваться 3-мя способами (а может и более): заголовки, в теле запроса, в адресе запроса
// 4.
import (
	"crypto/tls"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

func NewClient(logger *zap.Logger, proxyUrl *url.URL) *http.Client {
	var transport http.RoundTripper

	transport = &http.Transport{
		Proxy:           http.ProxyURL(proxyUrl),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if logger != nil {
		transport = &LoggingRoundTripper{transport, logger}
	}

	//transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return &http.Client{Transport: transport}
}
