// internal/policy/eval/eval.go
package eval

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type Compiled struct {
	program *vm.Program
	vars    []string
}

type MissingVariablesError struct {
	Vars []string
}

func (e *MissingVariablesError) Error() string {
	return fmt.Sprintf("missing variables: %s", strings.Join(e.Vars, ", "))
}

var compileCache sync.Map // map[string]*Compiled

func Compile(cond string) (*Compiled, error) {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return &Compiled{}, nil
	}

	if cached, ok := compileCache.Load(cond); ok {
		return cached.(*Compiled), nil
	}

	if err := Validate(cond); err != nil {
		return nil, err
	}

	program, err := expr.Compile(cond, expr.AsBool(), expr.AllowUndefinedVariables())
	if err != nil {
		return nil, err
	}

	compiled := &Compiled{
		program: program,
		vars:    extractVars(cond),
	}
	actual, _ := compileCache.LoadOrStore(cond, compiled)
	return actual.(*Compiled), nil
}

func Run(compiled *Compiled, vars map[string]any) (bool, error) {
	if compiled == nil || compiled.program == nil {
		return true, nil
	}

	missing := missingVars(compiled.vars, vars)
	if len(missing) > 0 {
		return false, &MissingVariablesError{Vars: missing}
	}

	out, err := expr.Run(compiled.program, vars)
	if err != nil {
		return false, err
	}

	b, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("cond must evaluate to bool (got %T)", out)
	}

	return b, nil
}

func Eval(cond string, vars map[string]any) (bool, error) {
	compiled, err := Compile(cond)
	if err != nil {
		return false, err
	}
	return Run(compiled, vars)
}

var identRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

func extractVars(cond string) []string {
	cond = stripQuoted(cond)
	matches := identRe.FindAllString(cond, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, name := range matches {
		switch name {
		case "true", "false":
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	sort.Strings(out)
	return out
}

func stripQuoted(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	quote := byte(0)
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		b.WriteByte(ch)
	}

	return b.String()
}

func missingVars(names []string, vars map[string]any) []string {
	if len(names) == 0 {
		return nil
	}

	out := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := vars[name]; !ok {
			out = append(out, name)
		}
	}
	return out
}
