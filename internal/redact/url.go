package redact

import (
	"net/url"
	"strings"
)

var sensitiveQueryKeys = map[string]struct{}{
	"token": {}, "key": {}, "password": {}, "secret": {}, "code": {},
	"session": {}, "auth": {}, "signature": {}, "sig": {}, "jwt": {},
	"access_token": {}, "refresh_token": {},
}

// URL strips userinfo and sensitive query params from a context URL.
// Returns empty strings if parsing fails.
func URL(raw string) (cleanURL, domain string) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", ""
	}
	u.User = nil
	q := u.Query()
	for k := range q {
		if _, bad := sensitiveQueryKeys[strings.ToLower(k)]; bad {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	host := strings.ToLower(u.Hostname())
	domain = strings.TrimPrefix(host, "www.")
	return u.String(), domain
}
