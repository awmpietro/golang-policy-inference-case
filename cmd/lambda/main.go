package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
	lambdatransport "github.com/awmpietro/golang-policy-inference-case/internal/transport/lambdatransport"
)

func main() {
	compiler := policy.NewCompiler()
	latencyObserver := policy.NewAsyncNodeLatencyObserver(policy.NewNodeLatencyLogger(log.Default()), 4096)
	defer latencyObserver.Close()
	engine := policy.NewEngine(
		policy.ExprEvaluator{},
		policy.WithNodeLatencyObserver(latencyObserver),
	)
	c := cache.NewInMemory(1024)

	svc := app.NewService(compiler, engine, c)
	h := lambdatransport.NewHandler(svc)

	lambda.Start(h.Infer)
}
