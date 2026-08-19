package shared

import (
	"net/url"
	"strings"
)

// SafeRedirect returns raw if it is a same-origin path suitable for a
// post-login redirect, otherwise it returns "". It rejects absolute URLs and
// protocol-relative paths to prevent open-redirect attacks.
func SafeRedirect(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Scheme != "" || u.Host != "" {
		return ""
	}
	return raw
}
