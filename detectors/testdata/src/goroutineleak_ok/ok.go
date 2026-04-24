package goroutineleaktest

import (
	"context"
	"time"
)

// Clean: listens on ctx.Done().
func okCtx(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}()
}

// Clean: reads from a channel the caller can close.
func okChan(stop <-chan struct{}) {
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
		}
	}()
}

// Clean: simple bounded goroutine; the loop has a condition.
func okBounded(n int) {
	go func() {
		for i := 0; i < n; i++ {
			_ = i
		}
	}()
}

// Clean: explicit break inside an unconditional for.
func okBreak() {
	go func() {
		for {
			break
		}
	}()
}

// Clean: return inside an unconditional for.
func okReturn() {
	go func() {
		for {
			return
		}
	}()
}

// Clean: naked channel send inside the loop keeps it observable.
func okSend(done chan<- int) {
	go func() {
		for {
			done <- 1
		}
	}()
}

// Clean: named function call (no inline FuncLit); we never flag those.
func workerForever() {
	for {
	}
}

func okNamedCall() {
	go workerForever()
}

// Clean: ctx.Done through method call.
func okCtxDone(ctx context.Context) {
	go func() {
		for {
			_ = ctx.Done()
		}
	}()
}
