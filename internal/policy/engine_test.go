// internal/policy/engine_test.go
package policy

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

type fakeEval struct {
	fn func(cond string, vars map[string]any) (bool, error)
}

func (f fakeEval) Eval(cond string, vars map[string]any) (bool, error) {
	return f.fn(cond, vars)
}

type spyLatencyObserver struct {
	nodes []string
	durs  []time.Duration
}

func (s *spyLatencyObserver) ObserveNodeLatency(nodeID string, duration time.Duration) {
	s.nodes = append(s.nodes, nodeID)
	s.durs = append(s.durs, duration)
}

func TestParseResult(t *testing.T) {
	a, err := ParseResult(`approved=true,segment=prime,score=720,label="ok"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 4 {
		t.Fatalf("expected 4 assignments, got %d", len(a))
	}
}

func TestEngine_Run_PicksFirstTrueEdge(t *testing.T) {
	// start -> A (true)
	// start -> B (also true)
	// regra: segue a PRIMEIRA true => A
	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{
			"start": {
				ID:     "start",
				Result: nil,
				Outgoing: []Edge{
					{To: "A", Cond: "c1"},
					{To: "B", Cond: "c2"},
				},
			},
			"A": {ID: "A", Result: []Assignment{{Key: "chosen", Value: "A"}}},
			"B": {ID: "B", Result: []Assignment{{Key: "chosen", Value: "B"}}},
		},
	}

	e := NewEngine(fakeEval{
		fn: func(cond string, vars map[string]any) (bool, error) {
			// ambos true
			return true, nil
		},
	})

	vars := map[string]any{}
	if err := e.Run(p, vars); err != nil {
		t.Fatal(err)
	}

	if vars["chosen"] != "A" {
		t.Fatalf("expected chosen=A, got %#v", vars["chosen"])
	}
}

func TestEngine_Run_NoEdgeMatchesTerminates(t *testing.T) {
	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{
			"start": {
				ID: "start",
				Outgoing: []Edge{
					{To: "A", Cond: "false"},
				},
			},
			"A": {ID: "A", Result: []Assignment{{Key: "x", Value: 1}}},
		},
	}

	e := NewEngine(fakeEval{
		fn: func(cond string, vars map[string]any) (bool, error) {
			return false, nil
		},
	})

	vars := map[string]any{"age": 20}
	if err := e.Run(p, vars); err != nil {
		t.Fatal(err)
	}

	// não visitou A, então não setou x
	if _, ok := vars["x"]; ok {
		t.Fatalf("expected x to be unset, got %#v", vars["x"])
	}
}

func TestEngine_Run_CondEvalErrorBubbles(t *testing.T) {
	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{
			"start": {
				ID: "start",
				Outgoing: []Edge{
					{To: "A", Cond: "bad"},
				},
			},
			"A": {ID: "A"},
		},
	}

	e := NewEngine(fakeEval{
		fn: func(cond string, vars map[string]any) (bool, error) {
			return false, fmt.Errorf("boom")
		},
	})

	vars := map[string]any{}
	err := e.Run(p, vars)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestEngine_Run_NoEdgeMatchedReportsMissingVars(t *testing.T) {
	compiler := NewCompiler()
	p, err := compiler.Compile(`digraph {
		start -> approved [cond="age>=18 && score>700"];
	}`)
	if err != nil {
		t.Fatal(err)
	}

	e := NewEngine(ExprEvaluator{})
	err = e.Run(p, map[string]any{"age": 20})
	if err == nil {
		t.Fatalf("expected error")
	}

	if !strings.Contains(err.Error(), `missing input vars [score]`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEngine_Run_ObservesNodeLatencyPerVisitedNode(t *testing.T) {
	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{
			"start": {
				ID: "start",
				Outgoing: []Edge{
					{To: "A", Cond: "true"},
				},
			},
			"A": {ID: "A", Result: []Assignment{{Key: "done", Value: true}}},
		},
	}

	observer := &spyLatencyObserver{}
	e := NewEngine(
		fakeEval{
			fn: func(cond string, vars map[string]any) (bool, error) {
				return true, nil
			},
		},
		WithNodeLatencyObserver(observer),
	)

	vars := map[string]any{}
	if err := e.Run(p, vars); err != nil {
		t.Fatal(err)
	}

	if len(observer.nodes) != 2 {
		t.Fatalf("expected 2 observed nodes, got %d", len(observer.nodes))
	}
	if observer.nodes[0] != "start" || observer.nodes[1] != "A" {
		t.Fatalf("unexpected nodes observed: %#v", observer.nodes)
	}
	for i, d := range observer.durs {
		if d < 0 {
			t.Fatalf("duration at %d is negative: %v", i, d)
		}
	}
}

func TestEngine_RunWithTrace_ReturnsVisitedPathAndSteps(t *testing.T) {
	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{
			"start": {
				ID: "start",
				Outgoing: []Edge{
					{To: "approved", Cond: "ok"},
					{To: "rejected", Cond: "no"},
				},
			},
			"approved": {ID: "approved", Result: []Assignment{{Key: "approved", Value: true}}},
			"rejected": {ID: "rejected", Result: []Assignment{{Key: "approved", Value: false}}},
		},
	}

	e := NewEngine(fakeEval{
		fn: func(cond string, vars map[string]any) (bool, error) {
			return cond == "ok", nil
		},
	})

	vars := map[string]any{}
	trace, err := e.RunWithTrace(p, vars)
	if err != nil {
		t.Fatal(err)
	}
	if vars["approved"] != true {
		t.Fatalf("expected approved=true, got %#v", vars["approved"])
	}
	if trace == nil {
		t.Fatalf("expected trace")
	}
	if trace.Terminated != "leaf" {
		t.Fatalf("expected termination leaf, got %q", trace.Terminated)
	}
	if len(trace.VisitedPath) != 2 || trace.VisitedPath[0] != "start" || trace.VisitedPath[1] != "approved" {
		t.Fatalf("unexpected visited path: %#v", trace.VisitedPath)
	}
	if len(trace.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(trace.Steps))
	}
	if trace.Steps[0].ChosenNext != "approved" {
		t.Fatalf("expected first step chosen next approved, got %#v", trace.Steps[0].ChosenNext)
	}
}

func TestEngine_RunWithTrace_NoEdgeMatchedTerminates(t *testing.T) {
	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{
			"start": {
				ID: "start",
				Outgoing: []Edge{
					{To: "A", Cond: "c1"},
				},
			},
			"A": {ID: "A"},
		},
	}

	e := NewEngine(fakeEval{
		fn: func(cond string, vars map[string]any) (bool, error) {
			return false, nil
		},
	})

	trace, err := e.RunWithTrace(p, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if trace == nil || trace.Terminated != "no_edge_matched" {
		t.Fatalf("expected no_edge_matched termination, got %#v", trace)
	}
	if len(trace.VisitedPath) != 1 || trace.VisitedPath[0] != "start" {
		t.Fatalf("unexpected visited path: %#v", trace.VisitedPath)
	}
}

func TestEngine_RunWithTrace_MissingVarsIncludesTraceOnError(t *testing.T) {
	compiler := NewCompiler()
	p, err := compiler.Compile(`digraph {
		start -> approved [cond="age>=18 && score>700"];
		approved [result="approved=true"];
	}`)
	if err != nil {
		t.Fatal(err)
	}

	e := NewEngine(ExprEvaluator{})
	trace, err := e.RunWithTrace(p, map[string]any{"age": 20})
	if err == nil {
		t.Fatalf("expected error")
	}
	if trace == nil {
		t.Fatalf("expected trace even on error")
	}
	if trace.Terminated != "error_no_edge_matched_missing_vars" {
		t.Fatalf("unexpected termination: %q", trace.Terminated)
	}
	if len(trace.Steps) == 0 || len(trace.Steps[0].Edges) == 0 {
		t.Fatalf("expected edge trace details, got %#v", trace.Steps)
	}
	if trace.Steps[0].Edges[0].Error == "" {
		t.Fatalf("expected edge error details in trace")
	}
}
