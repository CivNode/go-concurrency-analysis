package detectors

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// UnbufferedChanDeadlockAnalyzer flags the canonical unbuffered channel
// deadlock: a function declares `ch := make(chan T)` (unbuffered), sends on
// it, and never receives on it within the same function, and does not spawn
// a goroutine that receives on it. This is the "send on unbuffered chan with
// no receiver" pattern that blocks forever.
var UnbufferedChanDeadlockAnalyzer = &analysis.Analyzer{
	Name:     "UnbufferedChanDeadlock",
	Doc:      "reports sends on unbuffered channels with no paired receive in the same function",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runUnbufferedChanDeadlock,
}

func runUnbufferedChanDeadlock(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)}, func(n ast.Node) {
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
		analyzeFunctionForUnbufferedSend(pass, body)
	})
	return nil, nil
}

func analyzeFunctionForUnbufferedSend(pass *analysis.Pass, body *ast.BlockStmt) {
	// Find every `ch := make(chan T)` with no capacity or capacity 0.
	unbuffered := map[*types.Var]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		if len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		ce, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isUnbufferedMakeChan(ce) {
			return true
		}
		v, _ := pass.TypesInfo.ObjectOf(id).(*types.Var)
		if v != nil {
			unbuffered[v] = true
		}
		return true
	})
	if len(unbuffered) == 0 {
		return
	}

	sends := map[*types.Var][]ast.Node{}
	recvs := map[*types.Var]bool{}
	goroutineRecvs := map[*types.Var]bool{}

	var walk func(n ast.Node, inGoroutine bool)
	walk = func(n ast.Node, inGoroutine bool) {
		if n == nil {
			return
		}
		ast.Inspect(n, func(child ast.Node) bool {
			switch s := child.(type) {
			case *ast.GoStmt:
				if fl, ok := s.Call.Fun.(*ast.FuncLit); ok {
					walk(fl.Body, true)
				}
				return false
			case *ast.SendStmt:
				if v := channelVar(pass, s.Chan); v != nil && unbuffered[v] {
					if inGoroutine {
						// sends inside goroutine are also fine from the
						// caller's perspective, they unblock if caller
						// receives. Track them as receivable only if
						// caller receives; the caller path is the one
						// that matters.
						return true
					}
					sends[v] = append(sends[v], s)
				}
				return true
			case *ast.UnaryExpr:
				if s.Op.String() == "<-" {
					if v := channelVar(pass, s.X); v != nil && unbuffered[v] {
						if inGoroutine {
							goroutineRecvs[v] = true
						} else {
							recvs[v] = true
						}
					}
				}
				return true
			case *ast.RangeStmt:
				// `for v := range ch` counts as a receive on ch.
				if v := channelVar(pass, s.X); v != nil && unbuffered[v] {
					if inGoroutine {
						goroutineRecvs[v] = true
					} else {
						recvs[v] = true
					}
				}
			}
			return true
		})
	}
	walk(body, false)

	for v, nodes := range sends {
		if recvs[v] || goroutineRecvs[v] {
			continue
		}
		for _, n := range nodes {
			pass.Reportf(n.Pos(),
				"UnbufferedChanDeadlock: send on unbuffered channel %s has no paired receive in this function; this goroutine will block forever",
				v.Name())
		}
	}
}

func isUnbufferedMakeChan(ce *ast.CallExpr) bool {
	id, ok := ce.Fun.(*ast.Ident)
	if !ok || id.Name != "make" {
		return false
	}
	if len(ce.Args) == 0 {
		return false
	}
	if _, ok := ce.Args[0].(*ast.ChanType); !ok {
		return false
	}
	// No capacity arg = unbuffered. Capacity 0 literal = also unbuffered.
	if len(ce.Args) == 1 {
		return true
	}
	if lit, ok := ce.Args[1].(*ast.BasicLit); ok && lit.Value == "0" {
		return true
	}
	return false
}

func channelVar(pass *analysis.Pass, e ast.Expr) *types.Var {
	id, ok := e.(*ast.Ident)
	if !ok {
		return nil
	}
	v, _ := pass.TypesInfo.ObjectOf(id).(*types.Var)
	return v
}
