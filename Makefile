.PHONY: build-lambda sam

build-lambda:
	mkdir -p build
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/bootstrap ./cmd/lambda

sam: build-lambda
	sam local start-api