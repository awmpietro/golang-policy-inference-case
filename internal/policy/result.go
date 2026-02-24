// internal/policy/result.go
package policy

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseResult(raw string) ([]Assignment, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	out := make([]Assignment, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid assignment %q (expected key=value)", part)
		}

		key := strings.TrimSpace(kv[0])
		if key == "" {
			return nil, fmt.Errorf("empty key in assignment %q", part)
		}

		valRaw := strings.TrimSpace(kv[1])
		val := parseLiteral(valRaw)

		out = append(out, Assignment{Key: key, Value: val})
	}

	return out, nil
}

func parseLiteral(s string) any {
	s = strings.TrimSpace(s)

	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	if i, err := strconv.Atoi(s); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		if s[0] == '\'' {
			s = `"` + s[1:len(s)-1] + `"`
		}
		if unq, err := strconv.Unquote(s); err == nil {
			return unq
		}
	}

	return s
}
