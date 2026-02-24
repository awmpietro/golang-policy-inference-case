package policy

import "github.com/awmpietro/golang-policy-inference-case/internal/policy/eval"

type Policy struct {
	Start string
	Nodes map[string]*Node
}

type Node struct {
	ID       string
	Result   []Assignment
	Outgoing []Edge
}

type Edge struct {
	To           string
	Cond         string
	CompiledCond *eval.Compiled
}

type Assignment struct {
	Key   string
	Value any
}
