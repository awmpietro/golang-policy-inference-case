package main

import (
	"log"
	"net/http"

	"github.com/awmpietro/golang-policy-inference-case/internal/app"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
	httptransport "github.com/awmpietro/golang-policy-inference-case/internal/transport/http"
)

func main() {
	compiler := policy.NewCompiler()
	engine := policy.NewEngine(policy.ExprEvaluator{})
	c := cache.NewInMemory(1024)

	svc := app.NewService(compiler, engine, c)
	h := httptransport.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/infer", h.Infer)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
