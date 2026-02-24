// internal/app/service_test.go
package app

import (
	"fmt"
	"strings"
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

type fakeTraceEngine struct {
	fakeEngine
	trace *policy.ExecutionTrace
}

func (f *fakeTraceEngine) RunWithTrace(p *policy.Policy, vars map[string]any) (*policy.ExecutionTrace, error) {
	f.calls++
	if f.fakeEngine.fn != nil {
		if err := f.fakeEngine.fn(p, vars); err != nil {
			return f.trace, err
		}
	}
	return f.trace, nil
}

type fakeCache struct {
	calls   int
	lastKey string
}

func (c *fakeCache) GetOrCompute(dot string, fn func() (*policy.Policy, error)) (*policy.Policy, error) {
	c.calls++
	c.lastKey = dot
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

func TestService_InferWithTrace_UsesTraceEngine(t *testing.T) {
	comp := &fakeCompiler{
		p: &policy.Policy{Start: "start", Nodes: map[string]*policy.Node{"start": {ID: "start"}}},
	}
	eng := &fakeTraceEngine{
		fakeEngine: fakeEngine{
			fn: func(p *policy.Policy, vars map[string]any) error {
				vars["approved"] = true
				return nil
			},
		},
		trace: &policy.ExecutionTrace{
			StartNode:   "start",
			VisitedPath: []string{"start"},
			Terminated:  "leaf",
		},
	}

	s := NewService(comp, eng, &fakeCache{})

	out, trace, err := s.InferWithTrace("digraph { start [result=\"approved=true\"]; }", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if out["approved"] != true {
		t.Fatalf("expected approved=true in output")
	}
	if trace == nil || trace.StartNode != "start" {
		t.Fatalf("expected trace from engine, got %#v", trace)
	}
}

func TestService_InferWithTrace_FallsBackWhenEngineHasNoTrace(t *testing.T) {
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
	out, trace, err := s.InferWithTrace("digraph { start [result=\"approved=true\"]; }", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if out["approved"] != true {
		t.Fatalf("expected approved=true in output")
	}
	if trace != nil {
		t.Fatalf("expected nil trace fallback, got %#v", trace)
	}
}

func TestService_InferWithOptions_RequiresIDAndVersionTogether(t *testing.T) {
	comp := &fakeCompiler{
		p: &policy.Policy{Start: "start", Nodes: map[string]*policy.Node{"start": {ID: "start"}}},
	}
	eng := &fakeEngine{
		fn: func(p *policy.Policy, vars map[string]any) error {
			return nil
		},
	}
	c := &fakeCache{}
	s := NewService(comp, eng, c)

	_, _, err := s.InferWithOptions("digraph { start [result=\"\"]; }", map[string]any{}, InferOptions{PolicyID: "credit"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "policy_id and policy_version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestService_InferWithOptions_ReturnsPolicyInfoAndVersionedCacheKey(t *testing.T) {
	comp := &fakeCompiler{
		p: &policy.Policy{Start: "start", Nodes: map[string]*policy.Node{"start": {ID: "start"}}},
	}
	eng := &fakeEngine{
		fn: func(p *policy.Policy, vars map[string]any) error {
			vars["approved"] = true
			return nil
		},
	}
	c := &fakeCache{}
	s := NewService(comp, eng, c)

	out, info, err := s.InferWithOptions(
		"digraph { start [result=\"approved=true\"]; }",
		map[string]any{},
		InferOptions{PolicyID: "credit", PolicyVersion: "v1"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if out["approved"] != true {
		t.Fatalf("expected approved=true in output")
	}
	if info == nil || info.ID != "credit" || info.Version != "v1" || info.Hash == "" {
		t.Fatalf("unexpected policy info: %#v", info)
	}
	if !strings.HasPrefix(c.lastKey, "credit:v1:") {
		t.Fatalf("expected cache key with version prefix, got %q", c.lastKey)
	}
}
