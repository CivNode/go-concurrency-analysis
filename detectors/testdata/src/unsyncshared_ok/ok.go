package unsyncsharedtest

import (
	"sync"
	"sync/atomic"
)

var (
	counterOK int
	mu        sync.Mutex
)

// Clean: both writers hold the same mutex.
func incOK() {
	mu.Lock()
	counterOK++
	mu.Unlock()
}

func asyncIncOK() {
	go func() {
		mu.Lock()
		counterOK++
		mu.Unlock()
	}()
}

// Only one writer: no contention possible.
var soloVar int

func setSolo(n int) { soloVar = n }

// Clean: atomic writes from multiple functions including a goroutine.
var atomicCounter int64

func atomicInc() {
	atomic.AddInt64(&atomicCounter, 1)
}

func atomicIncAsync() {
	go func() {
		atomic.StoreInt64(&atomicCounter, 0)
	}()
}

// Clean: RWMutex counts as a guard too.
var rwCounter int
var rw sync.RWMutex

func rwInc() {
	rw.Lock()
	rwCounter++
	rw.Unlock()
}

func rwIncAsync() {
	go func() {
		rw.Lock()
		rwCounter++
		rw.Unlock()
	}()
}
