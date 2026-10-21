package normalize

import (
	"bytes"
	"unicode"
)

// Normalize performs: lower-case, collapse whitespace, strip punctuation optionally.
// Works on []byte to minimize allocations.
func Normalize(in []byte) []byte {
	if len(in) == 0 {
		return in
	}
	// 1. lower case (zero alloc if already lower, ToLower allocates anyway but we accept)
	lower := bytes.ToLower(in)

	// 2. optionally strip punctuation -> replace punct with space to keep token boundaries
	// We keep letters, numbers, space. This mirrors spec: "optionally strip punctuation"
	cleaned := bytes.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, lower)

	// 3. collapse whitespace: bytes.Fields + Join with single space (also trims)
	fields := bytes.Fields(cleaned)
	if len(fields) == 0 {
		return []byte{}
	}
	// bytes.Join allocates one slice; acceptable vs per-token alloc
	return bytes.Join(fields, []byte{' '})
}

func NormalizeString(s string) string {
	return string(Normalize([]byte(s)))
}
