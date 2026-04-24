package detectors

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// DeadlockAnalyzer flags reciprocal sync.Mutex lock order. For each goroutine
// spawned in the package we build the sequence of Mutex.Lock calls visible in
// the spawned body (by variable identity). If goroutine A locks m1 then m2 and
// goroutine B locks m2 then m1 on the same package-level Mutex pair, we flag
// both goroutines.
//
// Strictness: we only flag when the mutex operands resolve to the same
// package-level *sync.Mutex objects. Function parameters and locals are too
// ambiguous to prove a contradiction statically.
var DeadlockAnalyzer = &analysis.Analyzer{
	Name:     "Deadlock",
	Doc:      "reports reciprocal sync.Mutex lock orders between goroutines",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runDeadlock,
}

type lockSeq struct {
	vars []*types.Var // sequence of package-level Mutex vars locked
	pos  []ast.Node   // node where each lock is taken (for reporting)
	site ast.Node     // the go statement
}

func runDeadlock(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	var seqs []lockSeq
	insp.Preorder([]ast.Node{(*ast.GoStmt)(nil)}, func(n ast.Node) {
		goStmt := n.(*ast.GoStmt)
		seq := collectLockSequence(pass, goStmt)
		if len(seq.vars) >= 2 {
			seqs = append(seqs, seq)
		}
	})

	// Compare every pair. If seq A contains (m1, m2) in order and seq B
	// contains (m2, m1) in order (by package-level var identity), both are
	// reciprocal and we report both.
	reported := map[ast.Node]bool{}
	for i := range seqs {
		for j := i + 1; j < len(seqs); j++ {
			a, b := seqs[i], seqs[j]
			if m1, m2, ok := findReciprocalPair(a.vars, b.vars); ok {
				if !reported[a.site] {
					pass.Reportf(a.site.Pos(),
						"Deadlock: this goroutine locks %s then %s; another goroutine locks them in reverse order",
						m1.Name(), m2.Name())
					reported[a.site] = true
				}
				if !reported[b.site] {
					pass.Reportf(b.site.Pos(),
						"Deadlock: this goroutine locks %s then %s; another goroutine locks them in reverse order",
						m2.Name(), m1.Name())
					reported[b.site] = true
				}
			}
		}
	}
	return nil, nil
}

// collectLockSequence walks the body of a goroutine and records the order in
// which Mutex.Lock is called on package-level Mutex vars.
func collectLockSequence(pass *analysis.Pass, goStmt *ast.GoStmt) lockSeq {
	seq := lockSeq{site: goStmt}
	call := goStmt.Call
	var body *ast.BlockStmt
	if fl, ok := call.Fun.(*ast.FuncLit); ok {
		body = fl.Body
	}
	if body == nil {
		return seq
	}

	ast.Inspect(body, func(n ast.Node) bool {
		// Do not descend into nested func literals; they do not run
		// synchronously with the goroutine's lock order.
		if fl, ok := n.(*ast.FuncLit); ok {
			_ = fl
			return false
		}
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel == nil || sel.Sel.Name != "Lock" {
			return true
		}
		// Must be *sync.Mutex or sync.Mutex receiver.
		recvType := pass.TypesInfo.TypeOf(sel.X)
		if recvType == nil || !isSyncMutex(recvType) {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		obj := pass.TypesInfo.ObjectOf(id)
		v, ok := obj.(*types.Var)
		if !ok || v.Parent() == nil || v.Parent() != pass.Pkg.Scope() {
			// only package-level vars are tracked
			return true
		}
		seq.vars = append(seq.vars, v)
		seq.pos = append(seq.pos, ce)
		return true
	})
	return seq
}

func isSyncMutex(t types.Type) bool {
	// Accept both sync.Mutex and *sync.Mutex.
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "sync" && obj.Name() == "Mutex"
}

// findReciprocalPair looks for two vars m1, m2 such that a locks m1 before
// m2 (anywhere) and b locks m2 before m1 (anywhere). Returns m1, m2, true on
// match.
func findReciprocalPair(a, b []*types.Var) (*types.Var, *types.Var, bool) {
	before := func(seq []*types.Var, x, y *types.Var) bool {
		iX, iY := -1, -1
		for i, v := range seq {
			if v == x && iX < 0 {
				iX = i
			}
			if v == y && iY < 0 {
				iY = i
			}
		}
		return iX >= 0 && iY >= 0 && iX < iY
	}
	seen := map[*types.Var]bool{}
	for _, v := range a {
		seen[v] = true
	}
	for _, x := range a {
		for _, y := range b {
			if !seen[y] || x == y {
				continue
			}
			if before(a, x, y) && before(b, y, x) {
				return x, y, true
			}
		}
	}
	return nil, nil, false
}
