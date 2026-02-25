package policy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/awalterschulze/gographviz/ast"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy/eval"
)

type Compiler struct{}

func NewCompiler() *Compiler { return &Compiler{} }

// Compile pega o DOT cru, monta a Policy em memoria e já valida ciclo.
// Se a policy tiver ruim (parse ou semantica), da um failfast aqui pra nao estourar no runtime.
func (c *Compiler) Compile(dot string) (*Policy, error) {
	g, err := gographviz.ParseString(dot)
	if err != nil {
		return nil, fmt.Errorf("parse DOT: %w", err)
	}

	p := &Policy{
		Start: "start",
		Nodes: map[string]*Node{},
	}

	if err := walkStmtList(p, g.StmtList); err != nil {
		return nil, err
	}

	ensureNode(p, p.Start)
	if err := validateAcyclic(p); err != nil {
		return nil, err
	}

	return p, nil
}

func walkStmtList(p *Policy, stmts ast.StmtList) error {
	for _, st := range stmts {
		switch s := st.(type) {

		case *ast.NodeStmt:
			if err := applyNodeStmt(p, s); err != nil {
				return err
			}

		case ast.NodeStmt:
			tmp := s
			if err := applyNodeStmt(p, &tmp); err != nil {
				return err
			}

		case *ast.EdgeStmt:
			if err := applyEdgeStmt(p, s); err != nil {
				return err
			}

		case ast.EdgeStmt:
			tmp := s
			if err := applyEdgeStmt(p, &tmp); err != nil {
				return err
			}

		case *ast.SubGraph:
			if err := walkStmtList(p, s.StmtList); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyNodeStmt lê o result do nó (ex: approved=true,segment=prime) e guarda no modelo.
func applyNodeStmt(p *Policy, ns *ast.NodeStmt) error {
	if ns == nil || ns.NodeID == nil {
		return nil
	}

	id := string(ns.NodeID.GetID())
	node := ensureNode(p, id)

	attrs := ns.Attrs.GetMap()
	raw := strings.TrimSpace(unquote(attrs["result"]))

	assignments, err := ParseResult(raw)
	if err != nil {
		return fmt.Errorf("node %s invalid result: %w", id, err)
	}

	node.Result = assignments
	return nil
}

// applyEdgeStmt liga os nós e prepara a cond da aresta.
// A primeira aresta da chain recebe cond; as proximas ficam sem cond (sempre true).
func applyEdgeStmt(p *Policy, es *ast.EdgeStmt) error {
	if es == nil {
		return nil
	}

	from := string(es.Source.GetID())
	ensureNode(p, from)

	attrs := es.Attrs.GetMap()
	cond := strings.TrimSpace(unquote(attrs["cond"]))

	compiledCond, err := eval.Compile(cond)
	if err != nil {
		return fmt.Errorf("edge %s invalid cond: %w", from, err)
	}

	prev := from
	for i, rh := range es.EdgeRHS {
		if rh == nil {
			continue
		}

		to := string(rh.Destination.GetID())
		ensureNode(p, to)

		edgeCond := ""
		var edgeCompiled *eval.Compiled
		if i == 0 {
			edgeCond = cond
			edgeCompiled = compiledCond
		}

		p.Nodes[prev].Outgoing = append(p.Nodes[prev].Outgoing, Edge{
			To:           to,
			Cond:         edgeCond,
			CompiledCond: edgeCompiled,
		})

		prev = to
	}

	return nil
}

func ensureNode(p *Policy, id string) *Node {
	if n, ok := p.Nodes[id]; ok {
		return n
	}

	n := &Node{ID: id}
	p.Nodes[id] = n
	return n
}

func unquote(s string) string {
	s = strings.TrimSpace(s)

	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}

	return s
}

// validateAcyclic roda DFS simples com marcação de cor.
// Se achar back-edge, já devolve um erro mostrando o caminho do ciclo.
func validateAcyclic(p *Policy) error {
	const (
		unseen = iota
		visiting
		done
	)

	colors := make(map[string]int, len(p.Nodes))
	stack := make([]string, 0, len(p.Nodes))
	pos := make(map[string]int, len(p.Nodes))

	var dfs func(string) error
	dfs = func(id string) error {
		colors[id] = visiting
		pos[id] = len(stack)
		stack = append(stack, id)

		for _, edge := range p.Nodes[id].Outgoing {
			next := edge.To
			switch colors[next] {
			case unseen:
				if err := dfs(next); err != nil {
					return err
				}
			case visiting:
				start := pos[next]
				cycle := append([]string{}, stack[start:]...)
				cycle = append(cycle, next)
				return fmt.Errorf("policy graph contains cycle: %s", strings.Join(cycle, " -> "))
			}
		}

		stack = stack[:len(stack)-1]
		delete(pos, id)
		colors[id] = done
		return nil
	}

	ids := make([]string, 0, len(p.Nodes))
	for id := range p.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		if colors[id] != unseen {
			continue
		}
		if err := dfs(id); err != nil {
			return err
		}
	}

	return nil
}
