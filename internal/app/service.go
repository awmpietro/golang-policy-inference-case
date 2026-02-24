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

type Cache interface {
	GetOrCompute(dot string, fn func() (*policy.Policy, error)) (*policy.Policy, error)
}

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

func cloneMap(m map[string]any) map[string]any {
	n := make(map[string]any, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}
