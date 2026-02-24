package eval

import "testing"

func TestEval_ComparisonsAndLogic(t *testing.T) {
	vars := map[string]any{
		"age":   25,
		"score": 720,
	}

	ok, err := Eval(`age>=18 && score>700`, vars)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected true")
	}
}

func TestEval_StringEquality(t *testing.T) {
	vars := map[string]any{"segment": "prime"}

	ok, err := Eval(`segment=="prime"`, vars)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected true")
	}
}

func TestValidate_BlocksArithmetic(t *testing.T) {
	vars := map[string]any{"x": 1}

	_, err := Eval(`x+1==2`, vars)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidate_BlocksFunctionCall(t *testing.T) {
	vars := map[string]any{"x": 1}

	_, err := Eval(`len(x)==1`, vars)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidate_AllowsParentheses(t *testing.T) {
	vars := map[string]any{"a": true, "b": false, "c": true}

	ok, err := Eval(`a && (b || c)`, vars)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected true")
	}
}
