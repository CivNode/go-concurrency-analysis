package deadlocktest

import "sync"

var (
	mA sync.Mutex
	mB sync.Mutex
)

// Clean: both goroutines take the locks in the same order.
func ok() {
	go func() {
		mA.Lock()
		mB.Lock()
		mB.Unlock()
		mA.Unlock()
	}()

	go func() {
		mA.Lock()
		mB.Lock()
		mB.Unlock()
		mA.Unlock()
	}()
}

// Clean: only one mutex held at a time.
func okSingle() {
	go func() {
		mA.Lock()
		mA.Unlock()
	}()
	go func() {
		mB.Lock()
		mB.Unlock()
	}()
}

// Clean: lock taken on a local parameter (not a package var); we skip these
// entirely as we cannot prove identity.
type Locker struct {
	a, b sync.Mutex
}

func okLocal(l *Locker) {
	go func() {
		l.a.Lock()
		l.b.Lock()
		l.b.Unlock()
		l.a.Unlock()
	}()
	go func() {
		l.b.Lock()
		l.a.Lock()
		l.a.Unlock()
		l.b.Unlock()
	}()
}

// Clean: goroutine calls a named function, not a FuncLit.
func worker() {
	mA.Lock()
	mB.Lock()
	mB.Unlock()
	mA.Unlock()
}

func okNamed() {
	go worker()
}
