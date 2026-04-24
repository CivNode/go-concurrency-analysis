package detectors

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// GoroutineLeakAnalyzer flags `go func() { ... }()` whose body has no
// apparent exit path:
//   - no explicit return statement,
//   - no read from a `context.Context.Done()` channel (ctx.Done()),
//   - no channel send on any channel,
//   - no receive on a channel that could terminate the loop.
//
// The most common leak shape Training katas target is an infinite `for` loop
// that neither reads from a done channel nor sends/receives. That is what we
// flag. We deliberately do not flag plain `go worker()` calls: we cannot see
// the callee body reliably enough, and katas use inline closures.
var GoroutineLeakAnalyzer = &analysis.Analyzer{
	Name:     "GoroutineLeak",
	Doc:      "reports goroutines with no visible exit signal",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runGoroutineLeak,
}

func runGoroutineLeak(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder([]ast.Node{(*ast.GoStmt)(nil)}, func(n ast.Node) {
		goStmt := n.(*ast.GoStmt)
		fl, ok := goStmt.Call.Fun.(*ast.FuncLit)
		if !ok {
			return // only inline closures
		}
		if hasInfiniteLoopWithoutExit(pass, fl.Body) {
			pass.Reportf(goStmt.Pos(),
				"GoroutineLeak: goroutine has an infinite loop with no return, no context cancellation, and no channel operation; it cannot exit")
		}
	})
	return nil, nil
}

// hasInfiniteLoopWithoutExit reports whether body contains a top-level
// `for { ... }` (no condition, no post) whose body has no return, no ctx.Done
// receive, and no channel send/receive anywhere inside.
func hasInfiniteLoopWithoutExit(pass *analysis.Pass, body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	// Only look for a loop that is the last statement or the sole loop: a
	// goroutine with an unconditional infinite loop is the canonical leak.
	for _, stmt := range body.List {
		fs, ok := stmt.(*ast.ForStmt)
		if !ok {
			continue
		}
		if fs.Cond != nil || fs.Init != nil || fs.Post != nil {
			continue // not `for { }`
		}
		if loopHasExitSignal(pass, fs.Body) {
			continue
		}
		return true
	}
	return false
}

// loopHasExitSignal reports whether the given block performs any operation
// that could cause the goroutine to exit or be cancelled.
func loopHasExitSignal(pass *analysis.Pass, b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		if found {
			return false
		}
		switch s := n.(type) {
		case *ast.FuncLit:
			// skip nested closures; their signals do not terminate us
			return false
		case *ast.ReturnStmt:
			found = true
			return false
		case *ast.BranchStmt:
			if s.Tok.String() == "break" || s.Tok.String() == "goto" {
				// a `break` out of the infinite for exits the loop
				found = true
				return false
			}
		case *ast.SendStmt:
			found = true
			return false
		case *ast.UnaryExpr:
			if s.Op.String() == "<-" {
				found = true
				return false
			}
		case *ast.SelectStmt:
			// any select with a case is an exit signal (even default-only
			// selects can combine with another case; conservative accept)
			if len(s.Body.List) > 0 {
				found = true
				return false
			}
		case *ast.CallExpr:
			if isContextDoneCall(pass, s) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func isContextDoneCall(pass *analysis.Pass, ce *ast.CallExpr) bool {
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Done" {
		return false
	}
	recv := pass.TypesInfo.TypeOf(sel.X)
	if recv == nil {
		return false
	}
	// Unwrap pointer.
	if p, ok := recv.(*types.Pointer); ok {
		recv = p.Elem()
	}
	named, ok := recv.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "context" && obj.Name() == "Context"
}
