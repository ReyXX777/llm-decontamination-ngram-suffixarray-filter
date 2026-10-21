package config

import "runtime"

type Config struct {
	NGramSizes        []int   `json:"n_gram_sizes"`
	MinOverlapRatio   float64 `json:"min_overlap_ratio"`
	AbsoluteThreshold int     `json:"absolute_threshold"` // alternative to ratio
	ExactMatchMode    bool    `json:"exact_match_mode"`
	MinExactLength    int     `json:"min_exact_length"` // K
	NumWorkers        int     `json:"num_workers"`
	OutputCleanPath   string
	OutputReportPath  string
}

func DefaultConfig() *Config {
	return &Config{
		NGramSizes:        []int{8, 13, 20},
		MinOverlapRatio:   0.8,
		AbsoluteThreshold: 10,
		ExactMatchMode:    true,
		MinExactLength:    50,
		NumWorkers:        runtime.NumCPU(),
	}
}
