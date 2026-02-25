package app

import (
	"testing"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
	"github.com/awmpietro/golang-policy-inference-case/internal/policy/cache"
)

const benchPolicyDOT = `digraph Policy {
  start [result=""]
  approved [result="approved=true,segment=prime"]
  review [result="approved=false,segment=manual"]
  rejected [result="approved=false"]
  start -> approved [cond="age>=18 && score>700"]
  start -> review   [cond="age>=18 && score<=700"]
  start -> rejected [cond="age<18"]
}`

func benchmarkService() *Service {
	compiler := policy.NewCompiler()
	engine := policy.NewEngine(policy.ExprEvaluator{})
	c := cache.NewInMemory(1024)
	return NewService(compiler, engine, c)
}

func BenchmarkServiceInferCached(b *testing.B) {
	svc := benchmarkService()

	_, err := svc.Infer(benchPolicyDOT, map[string]any{"age": 25, "score": 720})
	if err != nil {
		b.Fatalf("warmup infer failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		input := map[string]any{"age": 25, "score": 720}
		_, err := svc.Infer(benchPolicyDOT, input)
		if err != nil {
			b.Fatalf("infer failed: %v", err)
		}
	}
}

func BenchmarkServiceInferCachedParallel(b *testing.B) {
	svc := benchmarkService()

	_, err := svc.Infer(benchPolicyDOT, map[string]any{"age": 25, "score": 720})
	if err != nil {
		b.Fatalf("warmup infer failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			input := map[string]any{"age": 25, "score": 720}
			_, err := svc.Infer(benchPolicyDOT, input)
			if err != nil {
				b.Fatalf("infer failed: %v", err)
			}
		}
	})
}
