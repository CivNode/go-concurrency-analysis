package detectors

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestUnsyncShared(t *testing.T) {
	dir := analysistest.TestData()
	analysistest.Run(t, dir, UnsyncSharedAnalyzer, "unsyncshared_ok", "unsyncshared_bad")
}
