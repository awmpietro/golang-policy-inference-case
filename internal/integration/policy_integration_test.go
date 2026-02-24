package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy"
)

func TestCompiler_Engine_Integration(t *testing.T) {
	dotPath := filepath.Join("..", "policy", "testdata", "simple.dot")
	dot, err := os.ReadFile(dotPath)
	if err != nil {
		t.Fatal(err)
	}

	compiler := policy.NewCompiler()
	engine := policy.NewEngine(policy.ExprEvaluator{})

	compiledPolicy, err := compiler.Compile(string(dot))
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]any{
		"age": 20,
	}

	err = engine.Run(compiledPolicy, vars)
	if err != nil {
		t.Fatal(err)
	}

	if vars["approved"] != true {
		t.Fatalf("expected approved=true, got %#v", vars["approved"])
	}
}
