package normalize

import (
	"bytes"
)

// Tokenize returns [][]byte tokens using whitespace split.
// Input should already be normalized.
// Uses bytes.Fields to avoid string allocations.
func Tokenize(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}
	return bytes.Fields(data)
}

// TokenizeString is helper for places where []string needed for map keys.
// Converts [][]byte -> []string with allocation (only at boundary).
func TokenizeString(data []byte) []string {
	bs := Tokenize(data)
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = string(b)
	}
	return out
}

// Preview returns first n chars safely for report
func Preview(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n])
}
