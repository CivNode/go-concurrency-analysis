package detectors

import "testing"

func TestAll(t *testing.T) {
	as := All()
	if len(as) < 5 {
		t.Fatalf("All() returned %d analyzers, want at least 5", len(as))
	}
	want := map[string]bool{
		"Deadlock": true, "GoroutineLeak": true, "UnsyncShared": true,
		"DeferInLoop": true, "UnbufferedChanDeadlock": true,
	}
	for _, a := range as {
		delete(want, a.Name)
	}
	if len(want) != 0 {
		t.Fatalf("All() missing analyzers: %v", want)
	}
}
