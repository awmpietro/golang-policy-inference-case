.PHONY: test test-race test-cover bench bench-profile load-test perf-check docker-build docker-run build-lambda sam

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

bench:
	go test -run '^$$' -bench BenchmarkServiceInfer -benchmem ./internal/app

bench-profile:
	go test -run '^$$' -bench BenchmarkServiceInferCached -cpuprofile cpu.out -memprofile mem.out ./internal/app
	go tool pprof -top cpu.out

load-test:
	go run ./cmd/loadtest -url http://localhost:8080/infer -rps 50 -duration 60s -workers 50

perf-check: bench load-test

docker-build:
	docker build -t policy-inference:local .

docker-run: docker-build
	docker run --rm --env-file .env -p 8080:8080 --name policy-inference policy-inference:local

build-lambda:
	mkdir -p build
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/bootstrap ./cmd/lambda

sam: build-lambda
	sam local start-api
