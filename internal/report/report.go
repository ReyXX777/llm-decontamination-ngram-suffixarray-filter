package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Summary struct {
	TotalDocs       int            `json:"total_docs"`
	CleanDocs       int            `json:"clean_docs"`
	Contaminated    int            `json:"contaminated"`
	MatchesByType   map[string]int `json:"matches_by_type"`
}

type Collector struct {
	mu       sync.Mutex
	summary  Summary
}

func NewCollector() *Collector {
	return &Collector{
		summary: Summary{MatchesByType: make(map[string]int)},
	}
}

func (c *Collector) AddReportLine(entry map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// naive counting
	if matches, ok := entry["matches"].([]interface{}); ok {
		for _, m := range matches {
			if s, ok := m.(string); ok {
				// prefix like n-gram-8 or exact-substring
				key := s
				if len(s) > 20 {
					key = s[:20]
				}
				c.summary.MatchesByType[key]++
			}
		}
	}
}

func SaveSummary(path string, clean, contam int) error {
	s := Summary{
		TotalDocs: clean + contam,
		CleanDocs: clean,
		Contaminated: contam,
		MatchesByType: map[string]int{},
	}
	f, err := os.Create(path + ".summary.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

func PrintSummary(clean, contam int) {
	total := clean + contam
	fmt.Printf("\n=== Contamination Report ===\n")
	fmt.Printf("Total scanned: %d\n", total)
	fmt.Printf("Clean: %d (%.2f%%)\n", clean, float64(clean)/float64(total+1)*100)
	fmt.Printf("Contaminated: %d (%.2f%%)\n", contam, float64(contam)/float64(total+1)*100)
}
