// Package jsonbuffer is a small utility package which helps identify JSON buffers.
package jsonbuffer

import "regexp"

var jsonBufferRegex = regexp.MustCompile(`^\s*\[?\s*{`)

// IsJSON scans the provided buffer, looking for an open brace indicating this is JSON.
//
// While a simple list like ["a", "b", "c"] is still a valid JSON,
// it does not really concern us when processing complex objects.
func IsJSON(buf []byte) bool {
	return jsonBufferRegex.Match(buf)
}
