package policy

import "fmt"

type Evaluator interface {
	Eval(cond string, vars map[string]any) (bool, error)
}

type Engine struct {
	eval Evaluator
}

func NewEngine(eval Evaluator) *Engine {
	return &Engine{eval: eval}
}

func (e *Engine) Run(p *Policy, vars map[string]any) error {
	if p == nil {
		return fmt.Errorf("policy is nil")
	}
	if p.Nodes == nil {
		return fmt.Errorf("policy nodes is nil")
	}

	start := p.Start
	if start == "" {
		start = "start"
	}

	current := start
	const maxSteps = 10_000

	for range maxSteps {
		node := p.Nodes[current]
		if node == nil {
			return fmt.Errorf("unknown node %q", current)
		}

		for _, a := range node.Result {
			vars[a.Key] = a.Value
		}

		if len(node.Outgoing) == 0 {
			return nil
		}

		found := false
		next := ""

		for _, edge := range node.Outgoing {
			ok, err := e.eval.Eval(edge.Cond, vars)
			if err != nil {
				return fmt.Errorf("cond eval failed at %s -> %s: %w", current, edge.To, err)
			}
			if ok {
				next = edge.To
				found = true
				break
			}
		}

		if !found {
			return nil
		}

		current = next
	}

	return fmt.Errorf("maxSteps exceeded (possible cycle or huge graph)")
}
