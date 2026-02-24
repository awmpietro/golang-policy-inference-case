package main

import (
	"log"
	"net/http"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
	httptransport "github.com/awmpietro/golang-policy-inference-case/internal/transport/httptransport"
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
	h := httptransport.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/infer", h.Infer)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
