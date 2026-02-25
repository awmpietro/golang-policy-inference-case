package eval

import (
	"fmt"
	"strings"
	"unicode"
)

func Validate(cond string) error {
	cond = strings.TrimSpace(cond)
	if cond == "" {
		return nil
	}

	illegalChars := []rune{'{', '}', '[', ']', ';', ':', '?', '@', '#', '$', '\\'}
	for _, ch := range illegalChars {
		if strings.ContainsRune(cond, ch) {
			return fmt.Errorf("illegal character %q", ch)
		}
	}

	if strings.Contains(cond, ".") {
		return fmt.Errorf("dot access is not allowed")
	}

	illegalOps := []string{"+", "-", "*", "/", "%"}
	for _, op := range illegalOps {
		if strings.Contains(cond, op) {
			return fmt.Errorf("arithmetic operator %q is not allowed", op)
		}
	}

	for i := 0; i < len(cond)-1; i++ {
		if cond[i] == '(' {
			j := i - 1
			for j >= 0 && unicode.IsSpace(rune(cond[j])) {
				j--
			}
			if j >= 0 && (unicode.IsLetter(rune(cond[j])) || cond[j] == '_') {
				k := j
				for k >= 0 && (unicode.IsLetter(rune(cond[k])) || unicode.IsDigit(rune(cond[k])) || cond[k] == '_') {
					k--
				}
				ident := strings.TrimSpace(cond[k+1 : j+1])
				if ident != "" {
					return fmt.Errorf("function calls are not allowed (found %q(...))", ident)
				}
			}
		}
	}

	return nil
}
