package covers

import "net/url"

// hosts is the allowlist of image origins the proxy will fetch. Restricting it
// keeps /__img from being usable as an open SSRF proxy.
var hosts = map[string]struct{}{
	"i.scdn.co":                 {},
	"mosaic.scdn.co":            {},
	"misc.scdn.co":              {},
	"lh3.googleusercontent.com": {},
}

// allowed validates and normalizes a raw cover URL, returning the URL to fetch
// and whether it is a permitted https origin.
func allowed(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" {
		return "", false
	}
	if _, ok := hosts[parsed.Hostname()]; !ok {
		return "", false
	}
	return parsed.String(), true
}
