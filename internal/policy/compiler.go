package policy

import (
	"fmt"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/awalterschulze/gographviz/ast"

	"github.com/awmpietro/golang-policy-inference-case/internal/policy/eval"
)

type Compiler struct{}

func NewCompiler() *Compiler { return &Compiler{} }

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

func applyEdgeStmt(p *Policy, es *ast.EdgeStmt) error {
	if es == nil {
		return nil
	}

	from := string(es.Source.GetID())
	ensureNode(p, from)

	attrs := es.Attrs.GetMap()
	cond := strings.TrimSpace(unquote(attrs["cond"]))

	if err := eval.Validate(cond); err != nil {
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
		if i == 0 {
			edgeCond = cond
		}

		p.Nodes[prev].Outgoing = append(p.Nodes[prev].Outgoing, Edge{
			To:   to,
			Cond: edgeCond,
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
