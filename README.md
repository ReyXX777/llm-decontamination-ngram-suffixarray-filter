# Contamination Analysis — N-gram + Suffix Array (Go)

Implementation of LLM training data decontamination to prevent benchmark leakage (MMLU, GSM8K, HumanEval etc.).

## Objective Formalized
```
D: set of training documents
B: set of benchmark examples
F_n = { Ngram_n(Tok(b),i) | b in B }
rho_n(d) = |{i | Ngram_n(Tok(d),i) in F_n}| / max(1, Len(Tok(d))-n+1)
ExactContam(d) <=> ∃ u (|u|>=K ∧ Substr(u,d) ∧ Substr(u,B))  [via Suffix Array]
Contaminated(d) <=> (∨_{n∈{8,13,20}} rho_n(d) >= θ) ∨ ExactContam(d)
D_clean = { d ∈ D | ¬Contaminated(d) }
```

## Architecture (Industry Standard)
```
contamination-detector/
├── cmd/detector/main.go          # CLI entrypoint, context cancellation
├── internal/
│   ├── config/config.go          # Tunable thresholds
│   ├── normalize/                # []byte-based normalization + tokenization
│   │   ├── normalize.go
│   │   └── tokenize.go
│   ├── detector/
│   │   ├── ngram.go              # map[K]struct{} with fixed arrays [8][13][20]string
│   │   ├── suffixarray.go        # Suffix Array exact substring
│   │   ├── detector.go           # Core CheckDoc logic
│   │   └── worker.go             # Worker pool + Fan-In collector
│   └── report/report.go          # Streaming aggregation
├── pkg/types/types.go            # Shared domain types
├── sample_data/                  # Tiny fixtures
└── go.mod
```

## Key Design Choices (per requirements)

1. **map[K]struct{} for Sets**: Zero-byte values, critical for millions of n-grams.
2. **Fixed-Size Arrays as Keys**: Slices not comparable. Use `[8]string`, `[13]string`, `[20]string`. Generic fallback uses joined string for other n.
3. **bufio.Scanner Streaming**: Never `os.ReadFile` on GB files. Line-by-line JSONL streaming.
4. **Worker Pools**: `NumWorkers = runtime.NumCPU()` goroutines scanning docs. Context-aware.
5. **[]byte Minimization**: `bytes.ToLower`, `bytes.Fields`, avoid string allocs in hot path.
6. **Channel Fan-In**: Workers -> results chan -> single collector goroutine writes reports, avoids race.
7. **json.NewEncoder Streaming**: Writes clean docs and report line-by-line, no giant slice.
8. **Suffix Array**: Self-contained implementation; replaceable with FM-index / Aho-Corasick lib in prod.
9. **pprof Ready**: `import _ "net/http/pprof"` hook, bench targets included.
10. **Context Cancellation**: Graceful Ctrl+C, no corrupted files.

## Usage
```bash
make run
# or
go run ./cmd/detector -train sample_data/train.jsonl -bench sample_data/benchmarks.jsonl -clean train_clean.jsonl -report contamination_report.json -workers 8 -k 50
```

## Thresholds
- `n_gram_sizes = [8,13,20]`
- `min_overlap_ratio = 0.8`
- `absolute_threshold = 10` n-grams
- `K = 50` chars for exact match
