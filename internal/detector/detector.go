package detector

import (
	"context"
	"fmt"

	"github.com/research/contamination-detector/internal/config"
	"github.com/research/contamination-detector/internal/normalize"
	"github.com/research/contamination-detector/pkg/types"
)

// Detector holds forbidden structures
type Detector struct {
	cfg          *config.Config
	ngrams       *NgramSet
	sa           *SuffixArray
	allBenchText string
}

func NewDetector(benchEntries []types.BenchmarkEntry, cfg *config.Config) *Detector {
	ngSet := NewNgramSet()
	var allTextsBuilder string
	// accumulate for SA
	var allBenchNormalized []string

	for _, be := range benchEntries {
		norm := normalize.Normalize([]byte(be.Text))
		if len(norm) == 0 {
			continue
		}
		normStr := string(norm)
		allBenchNormalized = append(allBenchNormalized, normStr)
		allTextsBuilder += normStr + "
"

		toks := normalize.TokenizeString(norm)
		for _, n := range cfg.NGramSizes {
			if len(toks) < n {
				continue
			}
			for i := 0; i <= len(toks)-n; i++ {
				ngSet.Add(toks[i : i+n])
			}
		}
	}

	var sa *SuffixArray
	if cfg.ExactMatchMode && len(allTextsBuilder) > 0 {
		sa = NewSuffixArray(allTextsBuilder)
	}

	return &Detector{
		cfg:          cfg,
		ngrams:       ngSet,
		sa:           sa,
		allBenchText: allTextsBuilder,
	}
}

// CheckResult for single doc
type CheckResult struct {
	DocID          int
	IsContaminated bool
	Matches        []string
	Preview        string
	CleanText      string
	OriginalText   string
}

// CheckDoc implements core logic from spec 3a + 3b
func (d *Detector) CheckDoc(ctx context.Context, docID int, rawText string) CheckResult {
	select {
	case <-ctx.Done():
		return CheckResult{DocID: docID}
	default:
	}

	normBytes := normalize.Normalize([]byte(rawText))
	if len(normBytes) == 0 {
		return CheckResult{DocID: docID, CleanText: rawText, OriginalText: rawText, Preview: ""}
	}
	tokens := normalize.TokenizeString(normBytes)

	isContam := false
	var matches []string

	// 3a N-gram overlap check
	for _, n := range d.cfg.NGramSizes {
		overlap, total := d.ngrams.CountOverlaps(tokens, n)
		if total == 0 {
			continue
		}
		ratio := float64(overlap) / float64(total)
		if ratio >= d.cfg.MinOverlapRatio || overlap >= d.cfg.AbsoluteThreshold {
			isContam = true
			matches = append(matches, fmt.Sprintf("n-gram-%d: %d/%d ratio=%.3f", n, overlap, total, ratio))
		}
	}

	// 3b Exact / long-substring via suffix array
	if d.cfg.ExactMatchMode && !isContam && d.sa != nil {
		queryStr := string(normBytes)
		if ok, hit := d.sa.HasLongMatch(queryStr, d.cfg.MinExactLength); ok {
			isContam = true
			matches = append(matches, fmt.Sprintf("exact-substring:%d:%s", d.cfg.MinExactLength, hit))
		}
	}

	preview := string(normBytes)
	if len(preview) > 200 {
		preview = preview[:200]
	}

	return CheckResult{
		DocID:          docID,
		IsContaminated: isContam,
		Matches:        matches,
		Preview:        preview,
		CleanText:      rawText,
		OriginalText:   rawText,
	}
}

func (d *Detector) Stats() (int, int, int) {
    if d.ngrams == nil {
        return 0,0,0
    }
    return d.ngrams.Stats()
}
func (d *Detector) AllBenchText() string {
    return d.allBenchText
}
