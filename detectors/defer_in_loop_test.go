package detectors

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestDeferInLoop(t *testing.T) {
	dir := analysistest.TestData()
	analysistest.Run(t, dir, DeferInLoopAnalyzer, "deferinloop_ok", "deferinloop_bad")
}
