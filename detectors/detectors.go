// Package detectors hosts the individual concurrency analyzers used by
// github.com/CivNode/go-concurrency-analysis.
//
// Each detector is an *analysis.Analyzer whose Name matches the Kind string it
// reports (e.g. "Deadlock", "DeferInLoop") so analysistest "// want" comments
// can reference detectors by Kind. Detectors deliberately stay conservative:
// the goal is zero false positives on realistic code, accepting misses on
// pathological constructs that cannot be proven statically.
package detectors

import "golang.org/x/tools/go/analysis"

// All returns every detector analyzer in a stable order.
func All() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		DeferInLoopAnalyzer,
	}
}
