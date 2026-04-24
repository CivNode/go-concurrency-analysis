package detectors

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// DeferInLoopAnalyzer flags `defer` statements whose nearest enclosing block
// is a for loop body. These accumulate per iteration and are a classic
// intermediate Go mistake.
var DeferInLoopAnalyzer = &analysis.Analyzer{
	Name:     "DeferInLoop",
	Doc:      "reports defer statements inside for loops",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runDeferInLoop,
}

func runDeferInLoop(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{(*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)}
	insp.Preorder(nodeFilter, func(n ast.Node) {
		var body *ast.BlockStmt
		switch fn := n.(type) {
		case *ast.FuncDecl:
			body = fn.Body
		case *ast.FuncLit:
			body = fn.Body
		}
		if body == nil {
			return
		}
		visitFunctionForDefer(pass, body)
	})
	return nil, nil
}

// visitFunctionForDefer walks the function body looking for `defer` statements
// that appear inside a for-loop body (direct or transitive, without crossing
// another function boundary).
func visitFunctionForDefer(pass *analysis.Pass, body *ast.BlockStmt) {
	var walk func(node ast.Node, inLoop bool)
	walk = func(node ast.Node, inLoop bool) {
		if node == nil {
			return
		}
		switch s := node.(type) {
		case *ast.FuncLit:
			// Nested function; its defers are bound to its own frame.
			return
		case *ast.DeferStmt:
			if inLoop {
				pass.Reportf(s.Pos(), "DeferInLoop: defer inside a for loop body accumulates until the enclosing function returns")
			}
			return
		case *ast.ForStmt:
			walk(s.Body, true)
			return
		case *ast.RangeStmt:
			walk(s.Body, true)
			return
		}
		ast.Inspect(node, func(child ast.Node) bool {
			if child == node {
				return true
			}
			switch child.(type) {
			case *ast.FuncLit, *ast.DeferStmt, *ast.ForStmt, *ast.RangeStmt:
				walk(child, inLoop)
				return false
			}
			return true
		})
	}
	walk(body, false)
}
