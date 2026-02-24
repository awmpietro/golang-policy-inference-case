package policy

import (
	"os"
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
}
