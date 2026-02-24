// internal/policy/compiler.go
package policy

import (
	"fmt"
	"strings"

	"github.com/awalterschulze/gographviz"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy/eval"
)

type Compiler struct{}

func NewCompiler() *Compiler { return &Compiler{} }

func (c *Compiler) Compile(dot string) (*Policy, error) {
	ast, err := gographviz.ParseString(dot)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DOT: %w", err)
	}

	g := gographviz.NewGraph()
	if err := gographviz.Analyse(ast, g); err != nil {
		return nil, fmt.Errorf("failed to analyze DOT: %w", err)
	}

	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{},
	}

	// 1) Nodes
	for _, n := range g.Nodes.Nodes {
		name := n.Name
		resultRaw := getAttr(n.Attrs, "result")

		assignments, err := ParseResult(resultRaw)
		if err != nil {
			return nil, fmt.Errorf("invalid result in node %q: %w", name, err)
		}

		p.Nodes[name] = &Node{
			ID:       name,
			Result:   assignments,
			Outgoing: []Edge{},
		}
	}

	if _, ok := p.Nodes[p.Start]; !ok {
		return nil, fmt.Errorf("missing %q node", p.Start)
	}

	orderedEdges, err := extractEdgesInTextOrder(dot)
	if err != nil {
		return nil, fmt.Errorf("failed to extract edge order from DOT: %w", err)
	}

	for _, e := range orderedEdges {
		fromNode, ok := p.Nodes[e.From]
		if !ok {
			return nil, fmt.Errorf("edge references unknown source node %q", e.From)
		}
		if _, ok := p.Nodes[e.To]; !ok {
			return nil, fmt.Errorf("edge references unknown destination node %q", e.To)
		}

		// cond pode ser vazia => trata como "true" (ou rejeita, você decide)
		cond := strings.TrimSpace(e.Cond)

		// valida cond (se vazia, seu Eval retorna true)
		if err := eval.Validate(cond); err != nil {
			return nil, fmt.Errorf("invalid cond on edge %s->%s: %w", e.From, e.To, err)
		}

		fromNode.Outgoing = append(fromNode.Outgoing, Edge{
			To:   e.To,
			Cond: cond,
		})
	}

	return p, nil
}

// getAttr lê atributo do Graphviz (normalmente vem com aspas).
func getAttr(attrs gographviz.Attrs, key string) string {
	val, ok := attrs[gographviz.Attr(key)]
	if !ok {
		return ""
	}

	val = strings.TrimSpace(val)

	// remove aspas "..."
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
		val = val[1 : len(val)-1]
	}

	return val
}
