package policy

import "github.com/awmpietro/golang-policy-inference-case/internal/policy/eval"

type ExprEvaluator struct{}

func (ExprEvaluator) Eval(cond string, vars map[string]any) (bool, error) {
	return eval.Eval(cond, vars)
}
