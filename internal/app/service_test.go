// internal/app/service_test.go
package app

import (
	"fmt"
	"testing"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
)

type fakeCompiler struct {
	calls int
	p     *policy.Policy
	err   error
}

func (f *fakeCompiler) Compile(dot string) (*policy.Policy, error) {
	f.calls++
	return f.p, f.err
}

type fakeEngine struct {
	calls int
	fn    func(p *policy.Policy, vars map[string]any) error
}

func (f *fakeEngine) Run(p *policy.Policy, vars map[string]any) error {
	f.calls++
	return f.fn(p, vars)
}

type fakeCache struct {
	calls int
	// simples: sempre chama compute (podemos melhorar no teste)
}

func (c *fakeCache) GetOrCompute(dot string, fn func() (*policy.Policy, error)) (*policy.Policy, error) {
	c.calls++
	return fn()
}

func TestService_Infer_ValidatesPolicyDot(t *testing.T) {
	s := NewService(&fakeCompiler{}, &fakeEngine{}, &fakeCache{})
	_, err := s.Infer("", map[string]any{"age": 1})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestService_Infer_ClonesInput(t *testing.T) {
	comp := &fakeCompiler{
		p: &policy.Policy{Start: "start", Nodes: map[string]*policy.Node{"start": {ID: "start"}}},
	}

	eng := &fakeEngine{
		fn: func(p *policy.Policy, vars map[string]any) error {
			vars["approved"] = true
			return nil
		},
	}

	s := NewService(comp, eng, &fakeCache{})

	in := map[string]any{"age": 20}
	out, err := s.Infer("digraph { start [result=\"\"]; }", in)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := in["approved"]; ok {
		t.Fatalf("expected input not to be mutated")
	}
	if out["approved"] != true {
		t.Fatalf("expected approved=true in output")
	}
}

func TestService_Infer_BubblesUpErrors(t *testing.T) {
	comp := &fakeCompiler{err: fmt.Errorf("compile fail")}
	eng := &fakeEngine{
		fn: func(p *policy.Policy, vars map[string]any) error {
			return nil
		},
	}
	s := NewService(comp, eng, &fakeCache{})

	_, err := s.Infer("x", map[string]any{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
