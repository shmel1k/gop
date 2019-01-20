all:
	GO111MODULE=on go mod vendor
	GO111MODULE=on go mod tidy
	go test -v ./... -cover -race

bench:
	go test ./... -bench=. -benchmem

bench_cpu:
	go test ./... -bench=. -cpuprofile=cpu.out

bench_mem:
	go test ./... -bench=. -memprofile=mem.out -benchmem

clean:
	rm *.out *.test
