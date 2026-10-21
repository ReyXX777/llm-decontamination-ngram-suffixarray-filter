BINARY=detector
CMD=./cmd/detector

.PHONY: build run test bench memprofile clean

build:
	go build -o bin/$(BINARY) $(CMD)

run: build
	./bin/$(BINARY) -train sample_data/train.jsonl -bench sample_data/benchmarks.jsonl -clean train_clean.jsonl -report contamination_report.json

test:
	go test ./... -v

bench:
	go test ./... -bench . -benchmem

memprofile:
	go test ./internal/detector -run ^$$ -bench BenchmarkScan -memprofile mem.out -cpuprofile cpu.out
	go tool pprof mem.out

clean:
	rm -rf bin/ train_clean.jsonl contamination_report.json mem.out cpu.out
