package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
	lambdatransport "github.com/awmpietro/golang-policy-inference-case/internal/transport/lambdatransport"
)

func main() {
	compiler := policy.NewCompiler()
	engine := policy.NewEngine(policy.ExprEvaluator{})
	c := cache.NewInMemory(1024)

	svc := app.NewService(compiler, engine, c)
	h := lambdatransport.NewHandler(svc)

	lambda.Start(h.Infer)
}
