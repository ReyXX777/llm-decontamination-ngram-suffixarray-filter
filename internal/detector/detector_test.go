package detector

import (
	"context"
	"testing"

	"github.com/research/contamination-detector/internal/config"
	"github.com/research/contamination-detector/pkg/types"
)

func TestContamination(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MinExactLength = 20
	bench := []types.BenchmarkEntry{
		{Benchmark: "MMLU", Text: "What is the capital of France? Paris"},
	}
	d := NewDetector(bench, cfg)
	ctx := context.Background()

	clean := d.CheckDoc(ctx, 1, "I love cooking pasta")
	if clean.IsContaminated {
		t.Errorf("expected clean doc to be clean")
	}

	contam := d.CheckDoc(ctx, 2, "What is the capital of France? Paris is the answer")
	if !contam.IsContaminated {
		t.Errorf("expected contaminated doc to be flagged")
	}
}

func BenchmarkScan(b *testing.B) {
	cfg := config.DefaultConfig()
	bench := []types.BenchmarkEntry{
		{Benchmark: "MMLU", Text: "The mitochondria is the powerhouse of the cell"},
	}
	d := NewDetector(bench, cfg)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.CheckDoc(ctx, i, "The mitochondria is the powerhouse of the cell and generates ATP")
	}
}
