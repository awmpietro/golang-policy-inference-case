package policy

import "testing"

func TestParseResult_TypesAndQuotes(t *testing.T) {
	assignments, err := ParseResult(`approved=true,segment="prime",score=720,ratio=1.5,label='ok'`)
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 5 {
		t.Fatalf("expected 5 assignments, got %d", len(assignments))
	}

	if v, ok := assignments[0].Value.(bool); !ok || !v {
		t.Fatalf("expected approved=true bool, got %#v", assignments[0].Value)
	}
	if v, ok := assignments[1].Value.(string); !ok || v != "prime" {
		t.Fatalf("expected segment=prime string, got %#v", assignments[1].Value)
	}
	if v, ok := assignments[2].Value.(int); !ok || v != 720 {
		t.Fatalf("expected score=720 int, got %#v", assignments[2].Value)
	}
	if v, ok := assignments[3].Value.(float64); !ok || v != 1.5 {
		t.Fatalf("expected ratio=1.5 float64, got %#v", assignments[3].Value)
	}
	if v, ok := assignments[4].Value.(string); !ok || v != "ok" {
		t.Fatalf("expected label=ok string, got %#v", assignments[4].Value)
	}
}

func TestParseResult_EmptyAndInvalid(t *testing.T) {
	assignments, err := ParseResult("")
	if err != nil {
		t.Fatal(err)
	}
	if assignments != nil {
		t.Fatalf("expected nil assignments for empty input, got %#v", assignments)
	}

	_, err = ParseResult("invalid")
	if err == nil {
		t.Fatalf("expected error for invalid assignment")
	}
}

func TestParseResult_QuotedBooleanRemainsString(t *testing.T) {
	assignments, err := ParseResult(`flag="true"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(assignments))
	}
	v, ok := assignments[0].Value.(string)
	if !ok || v != "true" {
		t.Fatalf("expected string true, got %#v", assignments[0].Value)
	}
}
