package detectors

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestGoroutineLeak(t *testing.T) {
	dir := analysistest.TestData()
	analysistest.Run(t, dir, GoroutineLeakAnalyzer, "goroutineleak_ok", "goroutineleak_bad")
}
