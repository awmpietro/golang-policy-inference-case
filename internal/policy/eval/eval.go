// internal/policy/eval/eval.go
package eval

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
)

func Eval(cond string, vars map[string]any) (bool, error) {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return true, nil
	}

	if err := Validate(cond); err != nil {
		return false, err
	}

	out, err := expr.Eval(cond, vars)
	if err != nil {
		return false, err
	}

	b, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("cond must evaluate to bool (got %T)", out)
	}

	return b, nil
}
