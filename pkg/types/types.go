package types

// Domain primitives
type BenchmarkEntry struct {
	Benchmark string `json:"benchmark"`
	Text      string `json:"text"`
}

type TrainDoc struct {
	ID   int    `json:"id,omitempty"`
	Text string `json:"text"`
}

// Output types
type CleanDoc struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

type ReportEntry struct {
	DocID         int               `json:"doc_id"`
	Matches       []string          `json:"matches"`
	Preview       string            `json:"preview"`
	BenchmarkHits map[string]int    `json:"benchmark_hits,omitempty"`
	IsContaminated bool             `json:"-"`
}
