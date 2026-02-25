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
	maxSteps        int
}

type EngineOption func(*Engine)

func WithNodeLatencyObserver(observer NodeLatencyObserver) EngineOption {
	return func(e *Engine) {
		e.latencyObserver = observer
	}
}

func WithMaxSteps(maxSteps int) EngineOption {
	return func(e *Engine) {
		if maxSteps > 0 {
			e.maxSteps = maxSteps
		}
	}
}

func NewEngine(eval Evaluator, opts ...EngineOption) *Engine {
	e := &Engine{
		eval:     eval,
		maxSteps: 10_000,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *Engine) Run(p *Policy, vars map[string]any) error {
	_, err := e.runInternal(p, vars, nil)
	return err
}

func (e *Engine) RunWithTrace(p *Policy, vars map[string]any) (*ExecutionTrace, error) {
	trace := &ExecutionTrace{}
	return e.runInternal(p, vars, trace)
}

func (e *Engine) runInternal(p *Policy, vars map[string]any, trace *ExecutionTrace) (*ExecutionTrace, error) {
	if p == nil {
		return trace, fmt.Errorf("policy is nil")
	}
	if p.Nodes == nil {
		return trace, fmt.Errorf("policy nodes is nil")
	}

	start := p.Start
	if start == "" {
		start = "start"
	}
	if trace != nil {
		trace.StartNode = start
	}

	current := start

	for range e.maxSteps {
		nodeStart := time.Now()
		step := TraceStep{NodeID: current}
		node := p.Nodes[current]
		if node == nil {
			duration := time.Since(nodeStart)
			e.observeNodeLatency(current, duration)
			step.DurationMicros = duration.Microseconds()
			appendTrace(trace, step)
			setTermination(trace, "error_unknown_node")
			return trace, fmt.Errorf("unknown node %q", current)
		}
		appendVisitedNode(trace, current)

		for _, a := range node.Result {
			vars[a.Key] = a.Value
		}

		if len(node.Outgoing) == 0 {
			duration := time.Since(nodeStart)
			e.observeNodeLatency(current, duration)
			step.DurationMicros = duration.Microseconds()
			appendTrace(trace, step)
			setTermination(trace, "leaf")
			return trace, nil
		}

		found := false
		next := ""
		errs := make([]string, 0, len(node.Outgoing))
		missingVars := map[string]struct{}{}
		edgeTraces := make([]EdgeTrace, 0, len(node.Outgoing))

		for _, edge := range node.Outgoing {
			ok, err := e.evalEdge(edge, vars)
			edgeTrace := EdgeTrace{To: edge.To, Cond: edge.Cond}
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s -> %s (%q): %v", current, edge.To, edge.Cond, err))
				edgeTrace.Error = err.Error()
				var mvErr *eval.MissingVariablesError
				if errors.As(err, &mvErr) {
					for _, name := range mvErr.Vars {
						missingVars[name] = struct{}{}
					}
				}
				edgeTraces = append(edgeTraces, edgeTrace)
				continue
			}
			if ok {
				next = edge.To
				found = true
				edgeTrace.Matched = true
				edgeTraces = append(edgeTraces, edgeTrace)
				break
			}
			edgeTraces = append(edgeTraces, edgeTrace)
		}
		step.Edges = edgeTraces

		if !found {
			duration := time.Since(nodeStart)
			e.observeNodeLatency(current, duration)
			step.DurationMicros = duration.Microseconds()
			appendTrace(trace, step)
			if len(errs) > 0 {
				if len(missingVars) > 0 {
					setTermination(trace, "error_no_edge_matched_missing_vars")
					return trace, fmt.Errorf("no edge matched at node %q: missing input vars [%s]; eval details: %s",
						current,
						joinSortedKeys(missingVars),
						strings.Join(errs, "; "),
					)
				}
				setTermination(trace, "error_no_edge_matched")
				return trace, fmt.Errorf("no edge matched at node %q: eval details: %s", current, strings.Join(errs, "; "))
			}
			setTermination(trace, "no_edge_matched")
			return trace, nil
		}

		step.ChosenNext = next
		duration := time.Since(nodeStart)
		e.observeNodeLatency(current, duration)
		step.DurationMicros = duration.Microseconds()
		appendTrace(trace, step)
		current = next
	}

	setTermination(trace, "error_max_steps")
	return trace, fmt.Errorf("maxSteps exceeded (possible cycle or huge graph)")
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

func appendVisitedNode(trace *ExecutionTrace, node string) {
	if trace == nil {
		return
	}
	trace.VisitedPath = append(trace.VisitedPath, node)
}

func appendTrace(trace *ExecutionTrace, step TraceStep) {
	if trace == nil {
		return
	}
	trace.Steps = append(trace.Steps, step)
}

func setTermination(trace *ExecutionTrace, terminated string) {
	if trace == nil {
		return
	}
	trace.Terminated = terminated
}
