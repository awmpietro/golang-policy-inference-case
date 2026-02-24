package policy

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy/eval"
)

type Evaluator interface {
	Eval(cond string, vars map[string]any) (bool, error)
}

type CompiledEvaluator interface {
	EvalCompiled(compiled *eval.Compiled, vars map[string]any) (bool, error)
}

type Engine struct {
	eval            Evaluator
	latencyObserver NodeLatencyObserver
}

type EngineOption func(*Engine)

func WithNodeLatencyObserver(observer NodeLatencyObserver) EngineOption {
	return func(e *Engine) {
		e.latencyObserver = observer
	}
}

func NewEngine(eval Evaluator, opts ...EngineOption) *Engine {
	e := &Engine{eval: eval}
	for _, opt := range opts {
		opt(e)
	}
	return e
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
		nodeStart := time.Now()
		node := p.Nodes[current]
		if node == nil {
			e.observeNodeLatency(current, time.Since(nodeStart))
			return fmt.Errorf("unknown node %q", current)
		}

		for _, a := range node.Result {
			vars[a.Key] = a.Value
		}

		if len(node.Outgoing) == 0 {
			e.observeNodeLatency(current, time.Since(nodeStart))
			return nil
		}

		found := false
		next := ""
		errs := make([]string, 0, len(node.Outgoing))
		missingVars := map[string]struct{}{}

		for _, edge := range node.Outgoing {
			ok, err := e.evalEdge(edge, vars)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s -> %s (%q): %v", current, edge.To, edge.Cond, err))
				var mvErr *eval.MissingVariablesError
				if errors.As(err, &mvErr) {
					for _, name := range mvErr.Vars {
						missingVars[name] = struct{}{}
					}
				}
				continue
			}
			if ok {
				next = edge.To
				found = true
				break
			}
		}

		if !found {
			e.observeNodeLatency(current, time.Since(nodeStart))
			if len(errs) > 0 {
				if len(missingVars) > 0 {
					return fmt.Errorf("no edge matched at node %q: missing input vars [%s]; eval details: %s",
						current,
						joinSortedKeys(missingVars),
						strings.Join(errs, "; "),
					)
				}
				return fmt.Errorf("no edge matched at node %q: eval details: %s", current, strings.Join(errs, "; "))
			}
			return nil
		}

		e.observeNodeLatency(current, time.Since(nodeStart))
		current = next
	}

	return fmt.Errorf("maxSteps exceeded (possible cycle or huge graph)")
}

func (e *Engine) observeNodeLatency(nodeID string, duration time.Duration) {
	if e.latencyObserver == nil {
		return
	}
	e.latencyObserver.ObserveNodeLatency(nodeID, duration)
}

func (e *Engine) evalEdge(edge Edge, vars map[string]any) (bool, error) {
	if edge.CompiledCond != nil {
		if ce, ok := e.eval.(CompiledEvaluator); ok {
			return ce.EvalCompiled(edge.CompiledCond, vars)
		}
	}
	return e.eval.Eval(edge.Cond, vars)
}

func joinSortedKeys(items map[string]struct{}) string {
	if len(items) == 0 {
		return ""
	}

	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
