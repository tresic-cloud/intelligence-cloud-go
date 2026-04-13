// Package transport provides internal HTTP transport primitives for the
// Intelligence Cloud Go SDK. This file implements sensitive-data redaction
// helpers used by the RoundTripper when recording span attributes and log
// lines.
package transport

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// sensitiveKeyPattern matches query-parameter or header names that
// contain sensitive material. Keys matching this pattern have their
// values replaced with "[REDACTED]" in telemetry output.
var sensitiveKeyPattern = regexp.MustCompile(`(?i)(token|secret|key|password)`)

// RedactedHeaders is the default set of HTTP header names whose values
// must never appear in spans, logs, or error payloads. The transport
// layer uses this list when building span attributes from request/response
// headers.
var RedactedHeaders = []string{
	"Authorization",
	"Cookie",
	"X-Api-Key",
	"X-Token",
}

// RedactURL returns the string representation of u with the values of
// query parameters whose key matches the sensitive-key pattern replaced
// by "[REDACTED]". The URL structure (scheme, host, path) is preserved.
// Returns an empty string if u is nil.
func RedactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	q := u.Query()
	redacted := false
	for k, vals := range q {
		if sensitiveKeyPattern.MatchString(k) {
			for i := range vals {
				vals[i] = "[REDACTED]"
			}
			q[k] = vals
			redacted = true
		}
	}
	if !redacted {
		return u.String()
	}
	cp := *u
	cp.RawQuery = q.Encode()
	return cp.String()
}

// RedactHeaders returns a copy of h containing only headers that are NOT
// in the RedactedHeaders set. Header names are compared case-insensitively.
// This is used by the transport layer to safely record headers in span
// attributes and log records.
func RedactHeaders(h http.Header) map[string]string {
	blocked := make(map[string]struct{}, len(RedactedHeaders))
	for _, name := range RedactedHeaders {
		blocked[strings.ToLower(name)] = struct{}{}
	}

	out := make(map[string]string, len(h))
	for name, vals := range h {
		if _, ok := blocked[strings.ToLower(name)]; ok {
			continue
		}
		out[name] = strings.Join(vals, ", ")
	}
	return out
}
