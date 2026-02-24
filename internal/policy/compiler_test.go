package policy

import (
	"os"
	"strings"
	"testing"
)

func TestCompiler_SimplePolicy(t *testing.T) {
	dot, err := os.ReadFile("testdata/simple.dot")
	if err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	p, err := compiler.Compile(string(dot))
	if err != nil {
		t.Fatal(err)
	}

	if len(p.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(p.Nodes))
	}

	start := p.Nodes["start"]
	if len(start.Outgoing) != 2 {
		t.Fatalf("expected 2 edges from start, got %d", len(start.Outgoing))
	}

	if start.Outgoing[0].CompiledCond == nil {
		t.Fatalf("expected first edge condition to be precompiled")
	}
}

func TestCompiler_RejectsCycle(t *testing.T) {
	compiler := NewCompiler()
	_, err := compiler.Compile(`digraph {
		start -> a [cond="x==1"];
		a -> b [cond="y==2"];
		b -> a [cond="z==3"];
	}`)
	if err == nil {
		t.Fatalf("expected cycle error")
	}
	if !strings.Contains(err.Error(), "contains cycle") {
		t.Fatalf("unexpected error: %v", err)
	}
}
