package httption

import "net/url"

type ProxyURLReplacer interface {
	ProxyURLReplace(u *url.URL)
}
