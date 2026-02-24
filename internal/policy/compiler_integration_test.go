package policy

import (
	"os"
	"testing"
)

func TestCompiler_Engine_Integration(t *testing.T) {
	dot, err := os.ReadFile("testdata/simple.dot")
	if err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	engine := NewEngine(ExprEvaluator{})

	policy, err := compiler.Compile(string(dot))
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]any{
		"age": 20,
	}

	err = engine.Run(policy, vars)
	if err != nil {
		t.Fatal(err)
	}

	if vars["approved"] != true {
		t.Fatalf("expected approved=true, got %#v", vars["approved"])
	}
}
