// Package logging provides slog setup and PII redaction helpers.
package logging

import (
	"regexp"
	"strings"
)

var (
	keyValueSensitive = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|authorization|cookie|card_number|cardnumber|cvv|ssn)"?\s*[:=]\s*"?([^",\s}&]+)`)
	bearerHeader      = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._\-]+`)
	panLike           = regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{1,7}\b`)
)

func maskValue(v string) string {
	switch {
	case len(v) <= 2:
		return "**"
	case len(v) <= 8:
		return v[:1] + strings.Repeat("*", len(v)-1)
	default:
		return v[:3] + strings.Repeat("*", 6)
	}
}

// Redact scrubs obvious secrets from a log line: key=value / JSON pairs for
// credential-ish keys, bearer tokens, and card-number-shaped digit runs.
// Best-effort by design — structured logs should avoid sensitive fields in
// the first place; this is defense-in-depth for error strings that embed
// request payloads.
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = keyValueSensitive.ReplaceAllStringFunc(s, func(m string) string {
		idx := strings.LastIndexAny(m, ":=")
		if idx < 0 {
			return m
		}
		return m[:idx+1] + `"` + maskValue(strings.Trim(m[idx+1:], `" `)) + `"`
	})
	s = bearerHeader.ReplaceAllString(s, "Bearer [REDACTED]")
	s = panLike.ReplaceAllStringFunc(s, func(m string) string {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, m)
		if luhnValid(digits) {
			return "[CARD]"
		}
		return m
	})
	return s
}

func luhnValid(digits string) bool {
	if len(digits) < 12 {
		return false
	}
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}
