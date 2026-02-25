package main

import (
	"log"
	"net/http"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/config"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
	httptransport "github.com/awmpietro/golang-policy-inference-case/internal/transport/httptransport"
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
	h := httptransport.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/infer", h.Infer)

	addr := cfg.HTTPAddr
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
