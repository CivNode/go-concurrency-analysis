package deferinlooptest

import "os"

func badRange(paths []string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close() // want "DeferInLoop"
		_ = f
	}
}

func badFor(n int) {
	for i := 0; i < n; i++ {
		f, err := os.Open("x")
		if err != nil {
			continue
		}
		defer f.Close() // want "DeferInLoop"
		_ = f
	}
}

// Nested block inside a for still flags.
func badNested(paths []string) {
	for _, p := range paths {
		if p != "" {
			f, _ := os.Open(p)
			defer f.Close() // want "DeferInLoop"
		}
	}
}
