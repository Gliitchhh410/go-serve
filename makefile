.PHONY: build test bench profile clean

build:
	go build -o server.exe ./cmd/tcplistener

test:
	go test ./...

bench:
	go test ./internal/request -bench=BenchmarkRequestFromReader -benchmem -benchtime=5s

bench-all:
	go test ./internal/request -bench=. -benchmem -benchtime=5s

fuzz:
	go test ./internal/request -fuzz=FuzzRequestFromReader -fuzztime=30s

cpu-profile:
	go test ./internal/request -bench=BenchmarkRequestFromReader -benchmem -benchtime=5s -cpuprofile=profiles/cpu.out

mem-profile:
	go test ./internal/request -bench=BenchmarkRequestFromReader -benchmem -benchtime=5s -memprofile=profiles/mem.out

pprof-cpu:
	go tool pprof -http=:8080 profiles/cpu.out

pprof-mem:
	go tool pprof -http=:8080 profiles/mem.out

clean:
	rm -f server.exe profiles/*.out