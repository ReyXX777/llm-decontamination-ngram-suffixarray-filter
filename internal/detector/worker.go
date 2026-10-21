package detector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/research/contamination-detector/internal/config"
	"github.com/research/contamination-detector/pkg/types"
)

// Job for worker pool
type Job struct {
	ID   int
	Text string
}

// RunPipeline streams train file, runs worker pool, fan-in to writers.
// Uses bufio.Scanner for memory efficiency, json.Encoder for streaming output.
func (d *Detector) RunPipeline(ctx context.Context, trainPath string, cleanOut io.Writer, reportOut io.Writer, cfg *config.Config) (int, int, error) {
	file, err := os.Open(trainPath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	jobs := make(chan Job, cfg.NumWorkers*2)
	results := make(chan CheckResult, cfg.NumWorkers*2)

	// context cancellation goroutine
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start workers (CPU-bound parallelism)
	var wg sync.WaitGroup
	for w := 0; w < cfg.NumWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// check context
				select {
				case <-ctx.Done():
					return
				default:
				}
				res := d.CheckDoc(ctx, job.ID, job.Text)
				select {
				case results <- res:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Collector goroutine - single writer to avoid race (Fan-In pattern)
	cleanEncoder := json.NewEncoder(cleanOut)
	reportEncoder := json.NewEncoder(reportOut)

	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	var cleanCount, contamCount int
	var mu sync.Mutex // for counts, encoders not thread-safe but only collector uses them

	go func() {
		defer collectorWg.Done()
		for res := range results {
			mu.Lock()
			if res.IsContaminated {
				contamCount++
				entry := types.ReportEntry{
					DocID:   res.DocID,
					Matches: res.Matches,
					Preview: res.Preview,
				}
				// best-effort report write
				_ = reportEncoder.Encode(entry)
			} else {
				cleanCount++
				clean := types.CleanDoc{ID: res.DocID, Text: res.OriginalText}
				_ = cleanEncoder.Encode(clean)
			}
			mu.Unlock()
		}
	}()

	// Producer: stream with bufio.Scanner (memory efficient)
	scanner := bufio.NewScanner(file)
	// allow large lines up to 10MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	docID := 0
producerLoop:
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			break producerLoop
		default:
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// parse JSONL train doc - tolerate both {"text":...} and raw text
		var td types.TrainDoc
		if err := json.Unmarshal(line, &td); err != nil {
			// fallback: treat line as raw text
			td.Text = string(line)
			td.ID = docID
		}
		if td.ID == 0 {
			td.ID = docID
		}
		// if text field empty but raw line is JSON with different key, try generic
		if td.Text == "" {
			var m map[string]interface{}
			if json.Unmarshal(line, &m) == nil {
				if t, ok := m["content"].(string); ok {
					td.Text = t
				} else if t, ok := m["document"].(string); ok {
					td.Text = t
				}
			}
			if td.Text == "" {
				td.Text = string(line)
			}
		}

		select {
		case jobs <- Job{ID: td.ID, Text: td.Text}:
			docID++
		case <-ctx.Done():
			break producerLoop
		}
	}
	close(jobs)
	wg.Wait()
	close(results)
	collectorWg.Wait()

	if err := scanner.Err(); err != nil {
		return cleanCount, contamCount, fmt.Errorf("scanner error: %w", err)
	}

	return cleanCount, contamCount, nil
}
