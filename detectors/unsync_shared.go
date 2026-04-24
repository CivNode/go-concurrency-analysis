package detectors

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// UnsyncSharedAnalyzer flags a package-level variable written from two or
// more functions where at least one write happens inside a goroutine body,
// and none of the writes are obviously guarded (mutex Lock visible in the
// same function, or sync/atomic call on that variable).
var UnsyncSharedAnalyzer = &analysis.Analyzer{
	Name:     "UnsyncShared",
	Doc:      "reports unsynchronized writes to package-level variables from multiple goroutines",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runUnsyncShared,
}

type writeSite struct {
	variable    *types.Var
	pos         token.Pos
	funcName    string
	inGoroutine bool
	guarded     bool
}

func runUnsyncShared(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	var writes []writeSite

	// Walk every top-level function and function literal. For each body,
	// record writes to package-level vars with whether the write happens
	// under a `go func(){ ... }()` and whether a mutex Lock is visible in
	// the enclosing function.
	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		fd := n.(*ast.FuncDecl)
		if fd.Body == nil {
			return
		}
		collectWrites(pass, fd.Name.Name, fd.Body, false, mutexLockInBody(pass, fd.Body), &writes)
	})

	// Group writes by var and decide.
	byVar := map[*types.Var][]writeSite{}
	for _, w := range writes {
		byVar[w.variable] = append(byVar[w.variable], w)
	}
	for v, ws := range byVar {
		if len(ws) < 2 {
			continue
		}
		funcs := map[string]bool{}
		anyGoroutine := false
		anyGuarded := false
		for _, w := range ws {
			funcs[w.funcName] = true
			if w.inGoroutine {
				anyGoroutine = true
			}
			if w.guarded {
				anyGuarded = true
			}
		}
		if len(funcs) < 2 || !anyGoroutine || anyGuarded {
			continue
		}
		for _, w := range ws {
			pass.Reportf(w.pos,
				"UnsyncShared: package-level variable %s is written from multiple functions including at least one goroutine, with no visible mutex or atomic guard",
				v.Name())
		}
	}
	return nil, nil
}

// collectWrites walks body. When it encounters a `go func(){...}()` it recurses
// with inGoroutine=true. Nested function literals outside a goroutine keep the
// caller's inGoroutine flag.
func collectWrites(pass *analysis.Pass, funcName string, body *ast.BlockStmt, inGoroutine, guarded bool, out *[]writeSite) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.GoStmt:
			if fl, ok := s.Call.Fun.(*ast.FuncLit); ok {
				g := guarded || mutexLockInBody(pass, fl.Body)
				collectWrites(pass, funcName, fl.Body, true, g, out)
			}
			return false
		case *ast.AssignStmt:
			if s.Tok != token.ASSIGN && s.Tok != token.ADD_ASSIGN && s.Tok != token.SUB_ASSIGN &&
				s.Tok != token.MUL_ASSIGN && s.Tok != token.QUO_ASSIGN {
				return true
			}
			for _, lhs := range s.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					obj := pass.TypesInfo.ObjectOf(id)
					if v, ok := obj.(*types.Var); ok && v.Parent() == pass.Pkg.Scope() {
						*out = append(*out, writeSite{
							variable: v, pos: id.Pos(), funcName: funcName,
							inGoroutine: inGoroutine, guarded: guarded,
						})
					}
				}
			}
		case *ast.IncDecStmt:
			if id, ok := s.X.(*ast.Ident); ok {
				obj := pass.TypesInfo.ObjectOf(id)
				if v, ok := obj.(*types.Var); ok && v.Parent() == pass.Pkg.Scope() {
					*out = append(*out, writeSite{
						variable: v, pos: id.Pos(), funcName: funcName,
						inGoroutine: inGoroutine, guarded: guarded,
					})
				}
			}
		case *ast.CallExpr:
			if isAtomicWrite(pass, s) {
				// Any atomic store to the variable counts as guarded; we
				// mark the matching write if it is an atomic on a pkg var.
				if v := atomicTargetVar(pass, s); v != nil && v.Parent() == pass.Pkg.Scope() {
					*out = append(*out, writeSite{
						variable: v, pos: s.Pos(), funcName: funcName,
						inGoroutine: inGoroutine, guarded: true,
					})
				}
			}
		}
		return true
	})
}

// mutexLockInBody scans a function body for any sync.Mutex.Lock call.
func mutexLockInBody(pass *analysis.Pass, body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			return true
		}
		if sel.Sel.Name != "Lock" && sel.Sel.Name != "RLock" {
			return true
		}
		t := pass.TypesInfo.TypeOf(sel.X)
		if t == nil {
			return true
		}
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
		}
		named, ok := t.(*types.Named)
		if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
			return true
		}
		if named.Obj().Pkg().Path() == "sync" &&
			(named.Obj().Name() == "Mutex" || named.Obj().Name() == "RWMutex") {
			found = true
			return false
		}
		return true
	})
	return found
}

func isAtomicWrite(pass *analysis.Pass, ce *ast.CallExpr) bool {
	sel, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgID, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName, ok := pass.TypesInfo.ObjectOf(pkgID).(*types.PkgName)
	if !ok {
		return false
	}
	if pkgName.Imported().Path() != "sync/atomic" {
		return false
	}
	switch sel.Sel.Name {
	case "StoreInt32", "StoreInt64", "StoreUint32", "StoreUint64", "StorePointer",
		"AddInt32", "AddInt64", "AddUint32", "AddUint64",
		"SwapInt32", "SwapInt64", "SwapUint32", "SwapUint64", "SwapPointer",
		"CompareAndSwapInt32", "CompareAndSwapInt64", "CompareAndSwapUint32",
		"CompareAndSwapUint64", "CompareAndSwapPointer":
		return true
	}
	return false
}

func atomicTargetVar(pass *analysis.Pass, ce *ast.CallExpr) *types.Var {
	if len(ce.Args) == 0 {
		return nil
	}
	arg := ce.Args[0]
	// atomic functions take &v as the first arg.
	if u, ok := arg.(*ast.UnaryExpr); ok && u.Op.String() == "&" {
		arg = u.X
	}
	id, ok := arg.(*ast.Ident)
	if !ok {
		return nil
	}
	v, _ := pass.TypesInfo.ObjectOf(id).(*types.Var)
	return v
}
