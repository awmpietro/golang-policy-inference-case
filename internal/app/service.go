// internal/app/service.go
package app

import (
	"fmt"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
)

type Compiler interface {
	Compile(dot string) (*policy.Policy, error)
}

type Engine interface {
	Run(p *policy.Policy, vars map[string]any) error
}

type TraceEngine interface {
	RunWithTrace(p *policy.Policy, vars map[string]any) (*policy.ExecutionTrace, error)
}

type Cache interface {
	GetOrCompute(dot string, fn func() (*policy.Policy, error)) (*policy.Policy, error)
}

type InferTrace = policy.ExecutionTrace

type Service struct {
	compiler Compiler
	engine   Engine
	cache    Cache
}

func NewService(compiler Compiler, engine Engine, cache Cache) *Service {
	return &Service{compiler: compiler, engine: engine, cache: cache}
}

// Infer compila (cacheado) e executa a policy, retornando input+inferidos.
// Não muta o input original.
func (s *Service) Infer(policyDOT string, input map[string]any) (map[string]any, error) {
	if policyDOT == "" {
		return nil, fmt.Errorf("policy_dot is required")
	}
	if input == nil {
		input = map[string]any{}
	}

	p, err := s.cache.GetOrCompute(policyDOT, func() (*policy.Policy, error) {
		return s.compiler.Compile(policyDOT)
	})
	if err != nil {
		return nil, err
	}

	out := cloneMap(input)
	if err := s.engine.Run(p, out); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) InferWithTrace(policyDOT string, input map[string]any) (map[string]any, *InferTrace, error) {
	if policyDOT == "" {
		return nil, nil, fmt.Errorf("policy_dot is required")
	}
	if input == nil {
		input = map[string]any{}
	}

	p, err := s.cache.GetOrCompute(policyDOT, func() (*policy.Policy, error) {
		return s.compiler.Compile(policyDOT)
	})
	if err != nil {
		return nil, nil, err
	}

	out := cloneMap(input)

	traceEngine, ok := s.engine.(TraceEngine)
	if !ok {
		if err := s.engine.Run(p, out); err != nil {
			return nil, nil, err
		}
		return out, nil, nil
	}

	trace, err := traceEngine.RunWithTrace(p, out)
	if err != nil {
		return nil, trace, err
	}

	return out, trace, nil
}

func cloneMap(m map[string]any) map[string]any {
	n := make(map[string]any, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}
