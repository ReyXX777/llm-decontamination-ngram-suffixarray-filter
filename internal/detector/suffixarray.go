package detector

import (
	"sort"
	"strings"
)

// SuffixArray is efficient lookup for exact / long-substring contamination.
// Production should replace with FM-index or Aho-Corasick library
// (e.g., github.com/huandu/go-suffixarray or github.com/cloudflare/ahocorasick)
// Here we implement SA with binary search for PhD assignment clarity.

type SuffixArray struct {
	text string
	sa   []int // sorted suffix starting indices
}

// NewSuffixArray builds SA for concatenated benchmark text.
// O(N log N log N) naive - ok for assignment, use SA-IS in prod for GB scale.
func NewSuffixArray(text string) *SuffixArray {
	n := len(text)
	sa := make([]int, n)
	for i := 0; i < n; i++ {
		sa[i] = i
	}
	sort.Slice(sa, func(i, j int) bool {
		// Compare suffixes text[sa[i]:] vs text[sa[j]:]
		return text[sa[i]:] < text[sa[j]:]
	})
	return &SuffixArray{text: text, sa: sa}
}

// lcp computes longest common prefix length of a and b, capped
func lcp(a, b string) int {
	m := len(a)
	if len(b) < m {
		m = len(b)
	}
	i := 0
	for i < m && a[i] == b[i] {
		i++
	}
	return i
}

// HasLongMatch checks if query contains any substring of sa.text with length >= minLen
// Implements ∃ u (|u|>=K ∧ Substr(u,query) ∧ Substr(u,B))
func (s *SuffixArray) HasLongMatch(query string, minLen int) (bool, string) {
	if len(s.text) == 0 || len(query) < minLen {
		return false, ""
	}
	// For each suffix of query, binary search in SA
	// Instead of O(|query| * log N) with LCP check of neighbors
	for start := 0; start <= len(query)-minLen; start++ {
		qSuffix := query[start:]
		// lower bound binary search: first SA suffix >= qSuffix
		idx := sort.Search(len(s.sa), func(i int) bool {
			// sa suffix string
			suf := s.text[s.sa[i]:]
			// compare suf >= qSuffix lexicographically for prefix
			// We use strings.Compare but limit to len qSuffix for lower bound logic
			return suf >= qSuffix
		})
		// Check idx and idx-1 for LCP >= minLen
		candidates := []int{}
		if idx < len(s.sa) {
			candidates = append(candidates, idx)
		}
		if idx > 0 {
			candidates = append(candidates, idx-1)
		}
		if idx+1 < len(s.sa) {
			candidates = append(candidates, idx+1)
		}
		for _, ci := range candidates {
			benchSuffix := s.text[s.sa[ci]:]
			common := lcp(qSuffix, benchSuffix)
			if common >= minLen {
				// Return matched substring preview
				end := start + common
				if end > start+100 {
					end = start + 100
				}
				return true, query[start:end]
			}
		}
	}
	return false, ""
}

// Alternative exact check using strings.Contains for small benchmarks (fallback)
func (s *SuffixArray) ContainsExact(sub string) bool {
	return strings.Contains(s.text, sub)
}
