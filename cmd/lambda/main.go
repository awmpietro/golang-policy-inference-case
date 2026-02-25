package main

import (
	"log"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/config"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
	lambdatransport "github.com/awmpietro/golang-policy-inference-case/internal/transport/lambdatransport"
)

func main() {
	cfg := config.Load()

	compiler := policy.NewCompiler()
	latencyObserver := policy.NewAsyncNodeLatencyObserver(policy.NewNodeLatencyLogger(log.Default()), cfg.ObsBuffer)
	defer latencyObserver.Close()
	engine := policy.NewEngine(
		policy.ExprEvaluator{},
		policy.WithNodeLatencyObserver(latencyObserver),
		policy.WithMaxSteps(cfg.PolicyMaxSteps),
	)
	c := cache.NewInMemory(cfg.CacheMaxItems)

	svc := app.NewService(compiler, engine, c)
	h := lambdatransport.NewHandler(svc)

	lambda.Start(h.Infer)
}
