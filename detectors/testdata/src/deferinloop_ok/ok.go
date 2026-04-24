package deferinlooptest

import "os"

// Clean: defer at function scope only.
func cleanOne() error {
	f, err := os.Open("x")
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

// Clean: defer inside an inner func literal called per iteration. The defer
// is scoped to the literal, not the loop.
func cleanTwo(paths []string) {
	for _, p := range paths {
		func(p string) {
			f, err := os.Open(p)
			if err != nil {
				return
			}
			defer f.Close()
			_ = f
		}(p)
	}
}

// Clean: for with a defer that escapes the loop via a nested func call path
// (we do not flag inner closures).
func cleanThree() {
	for i := 0; i < 10; i++ {
		go func() {
			defer println("done")
			_ = i
		}()
	}
}

// Clean: defer at function scope after a for loop.
func cleanFour() {
	for i := 0; i < 2; i++ {
		_ = i
	}
	f, _ := os.Open("x")
	defer f.Close()
}
