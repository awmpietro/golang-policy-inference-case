// internal/app/service.go
package app

import (
	"crypto/sha256"
	"encoding/hex"
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

type InferOptions struct {
	PolicyID      string
	PolicyVersion string
}

type PolicyInfo struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Hash    string `json:"hash"`
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
	out, _, err := s.InferWithOptions(policyDOT, input, InferOptions{})
	return out, err
}

func (s *Service) InferWithOptions(policyDOT string, input map[string]any, opts InferOptions) (map[string]any, *PolicyInfo, error) {
	p, out, info, err := s.prepare(policyDOT, input, opts)
	if err != nil {
		return nil, nil, err
	}

	if err := s.engine.Run(p, out); err != nil {
		return nil, info, err
	}

	return out, info, nil
}

func (s *Service) InferWithTrace(policyDOT string, input map[string]any) (map[string]any, *InferTrace, error) {
	out, trace, _, err := s.InferWithTraceAndOptions(policyDOT, input, InferOptions{})
	return out, trace, err
}

func (s *Service) InferWithTraceAndOptions(policyDOT string, input map[string]any, opts InferOptions) (map[string]any, *InferTrace, *PolicyInfo, error) {
	p, out, info, err := s.prepare(policyDOT, input, opts)
	if err != nil {
		return nil, nil, nil, err
	}

	traceEngine, ok := s.engine.(TraceEngine)
	if !ok {
		if err := s.engine.Run(p, out); err != nil {
			return nil, nil, info, err
		}
		return out, nil, info, nil
	}

	trace, err := traceEngine.RunWithTrace(p, out)
	if err != nil {
		return nil, trace, info, err
	}

	return out, trace, info, nil
}

func (s *Service) prepare(policyDOT string, input map[string]any, opts InferOptions) (*policy.Policy, map[string]any, *PolicyInfo, error) {
	if policyDOT == "" {
		return nil, nil, nil, fmt.Errorf("policy_dot is required")
	}
	if input == nil {
		input = map[string]any{}
	}
	if (opts.PolicyID == "") != (opts.PolicyVersion == "") {
		return nil, nil, nil, fmt.Errorf("policy_id and policy_version must be provided together")
	}

	policyHash := hash(policyDOT)
	var info *PolicyInfo
	if opts.PolicyID != "" {
		info = &PolicyInfo{
			ID:      opts.PolicyID,
			Version: opts.PolicyVersion,
			Hash:    policyHash,
		}
	}

	p, err := s.cache.GetOrCompute(cacheKey(opts, policyHash), func() (*policy.Policy, error) {
		return s.compiler.Compile(policyDOT)
	})
	if err != nil {
		return nil, nil, nil, err
	}

	out := cloneMap(input)
	return p, out, info, nil
}

func cloneMap(m map[string]any) map[string]any {
	n := make(map[string]any, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}

func cacheKey(opts InferOptions, policyHash string) string {
	if opts.PolicyID == "" {
		return policyHash
	}
	return fmt.Sprintf("%s:%s:%s", opts.PolicyID, opts.PolicyVersion, policyHash)
}

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
