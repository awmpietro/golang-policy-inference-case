package policy

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
	To   string
	Cond string
}

type Assignment struct {
	Key   string
	Value any
}
