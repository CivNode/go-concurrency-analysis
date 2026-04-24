package detectors

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestDeadlock(t *testing.T) {
	dir := analysistest.TestData()
	analysistest.Run(t, dir, DeadlockAnalyzer, "deadlock_ok", "deadlock_bad")
}
