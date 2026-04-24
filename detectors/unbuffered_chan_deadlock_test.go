package detectors

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestUnbufferedChanDeadlock(t *testing.T) {
	dir := analysistest.TestData()
	analysistest.Run(t, dir, UnbufferedChanDeadlockAnalyzer, "unbufferedchan_ok", "unbufferedchan_bad")
}
