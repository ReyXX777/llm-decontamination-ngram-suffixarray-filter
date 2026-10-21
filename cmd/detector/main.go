package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/research/contamination-detector/internal/config"
	"github.com/research/contamination-detector/internal/detector"
	"github.com/research/contamination-detector/internal/report"
	"github.com/research/contamination-detector/pkg/types"
)

func main() {
	// CLI flags - industry standard
	trainPath := flag.String("train", "sample_data/train.jsonl", "Path to train docs JSONL (streamed)")
	benchPath := flag.String("bench", "sample_data/benchmarks.jsonl", "Path to benchmarks JSONL")
	cleanPath := flag.String("clean", "train_clean.jsonl", "Output clean docs JSONL")
	reportPath := flag.String("report", "contamination_report.json", "Output contamination report JSONL")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of workers")
	minOverlap := flag.Float64("overlap", 0.8, "Min overlap ratio")
	absThresh := flag.Int("abs", 10, "Absolute n-gram count threshold")
	k := flag.Int("k", 50, "Min exact substring length K")
	exact := flag.Bool("exact", true, "Enable exact match via suffix array")
	flag.Parse()

	cfg := config.DefaultConfig()
	cfg.NumWorkers = *workers
	cfg.MinOverlapRatio = *minOverlap
	cfg.AbsoluteThreshold = *absThresh
	cfg.MinExactLength = *k
	cfg.ExactMatchMode = *exact
	cfg.OutputCleanPath = *cleanPath
	cfg.OutputReportPath = *reportPath

	// Context with cancellation for long-running jobs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n[Interrupt] Cancelling... flushing buffers")
		cancel()
	}()

	start := time.Now()

	// 1. Load benchmarks streaming with bufio.Scanner
	fmt.Printf("Loading benchmarks from %s...\n", *benchPath)
	benchEntries, err := loadBenchmarks(*benchPath)
	if err != nil {
		log.Fatalf("Failed to load benchmarks: %v", err)
	}
	fmt.Printf("Loaded %d benchmark examples\n", len(benchEntries))
	n8, n13, n20 := 0,0,0
	_ = n8
	_ = n13
	_ = n20

	// 2. Build detector (forbidden n-grams + suffix array)
	fmt.Println("Building forbidden n-gram sets and suffix array...")
	d := detector.NewDetector(benchEntries, cfg)
	s8, s13, s20 := d.Stats()
	fmt.Printf("Forbidden n-grams: 8:%d 13:%d 20:%d\n", s8, s13, s20)
	fmt.Printf("Suffix Array text length: %d chars\n", len(d.AllBenchText()))

	// 3. Open output files with streaming encoder
	cleanFile, err := os.Create(*cleanPath)
	if err != nil {
		log.Fatalf("Create clean file: %v", err)
	}
	defer cleanFile.Close()

	reportFile, err := os.Create(*reportPath)
	if err != nil {
		log.Fatalf("Create report file: %v", err)
	}
	defer reportFile.Close()

	// Buffered writers for performance
	cleanBuf := bufio.NewWriterSize(cleanFile, 4*1024*1024)
	reportBuf := bufio.NewWriterSize(reportFile, 4*1024*1024)
	defer cleanBuf.Flush()
	defer reportBuf.Flush()

	// 4. Run pipeline
	fmt.Printf("Scanning train docs from %s with %d workers...\n", *trainPath, cfg.NumWorkers)
	cleanCount, contamCount, err := d.RunPipeline(ctx, *trainPath, cleanBuf, reportBuf, cfg)
	if err != nil {
		log.Fatalf("Pipeline error: %v", err)
	}

	elapsed := time.Since(start)
	report.PrintSummary(cleanCount, contamCount)
	fmt.Printf("Time: %s\n", elapsed)
	fmt.Printf("Clean saved to %s\n", *cleanPath)
	fmt.Printf("Report saved to %s\n", *reportPath)

	// Save summary json
	_ = report.SaveSummary(*reportPath, cleanCount, contamCount)
}

// loadBenchmarks streams file with bufio.Scanner (requirement)
func loadBenchmarks(path string) ([]types.BenchmarkEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []types.BenchmarkEntry
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		var be types.BenchmarkEntry
		if err := json.Unmarshal([]byte(line), &be); err != nil {
			// tolerate {"name":..., "example":...} etc
			var generic map[string]string
			if json.Unmarshal([]byte(line), &generic) == nil {
				if t, ok := generic["text"]; ok {
					be.Text = t
					be.Benchmark = generic["benchmark"]
				} else if t, ok := generic["content"]; ok {
					be.Text = t
				} else {
					be.Text = line
				}
			} else {
				be.Text = line
			}
		}
		if be.Text != "" {
			entries = append(entries, be)
		}
	}
	return entries, scanner.Err()
}

// Extend Detector with Stats helpers via wrapper - avoid import cycle, add methods in detector package

