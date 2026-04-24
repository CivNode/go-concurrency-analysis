package goroutineleaktest

import "time"

// Infinite loop with no context, no channel op, no return.
func leak() {
	go func() { // want "GoroutineLeak"
		for {
			time.Sleep(time.Second)
		}
	}()
}

// Same shape, different body: still a leak.
func leakCounter() {
	var n int
	go func() { // want "GoroutineLeak"
		for {
			n++
		}
	}()
	_ = n
}
