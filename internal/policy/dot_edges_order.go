package policy

import (
	"fmt"
	"regexp"
	"strings"
)

type edgeSpec struct {
	From string
	To   string
	Cond string
}

func splitStatements(dot string) []string {
	var out []string
	var b strings.Builder
	inQuotes := false
	escape := false

	for _, r := range dot {
		if escape {
			b.WriteRune(r)
			escape = false
			continue
		}
		if r == '\\' && inQuotes {
			b.WriteRune(r)
			escape = true
			continue
		}
		if r == '"' {
			inQuotes = !inQuotes
			b.WriteRune(r)
			continue
		}
		if r == ';' && !inQuotes {
			s := strings.TrimSpace(b.String())
			if s != "" {
				out = append(out, s)
			}
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	if s := strings.TrimSpace(b.String()); s != "" {
		out = append(out, s)
	}
	return out
}

var edgeStmtRe = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s*->\s*([A-Za-z_][A-Za-z0-9_]*)\s*(\[(.*)\])?\s*$`)
var condRe = regexp.MustCompile(`cond\s*=\s*"([^"]*)"`)

func extractEdgesInTextOrder(dot string) ([]edgeSpec, error) {
	stmts := splitStatements(dot)
	out := make([]edgeSpec, 0)

	for _, s := range stmts {
		if !strings.Contains(s, "->") {
			continue
		}

		m := edgeStmtRe.FindStringSubmatch(s)
		if m == nil {
			return nil, fmt.Errorf("unsupported edge statement: %q", s)
		}

		from := m[1]
		to := m[2]
		attrs := ""
		if len(m) >= 5 {
			attrs = m[4]
		}

		cond := ""
		if attrs != "" {
			cm := condRe.FindStringSubmatch(attrs)
			if cm != nil {
				cond = strings.TrimSpace(cm[1])
			}
		}

		out = append(out, edgeSpec{From: from, To: to, Cond: cond})
	}

	return out, nil
}
