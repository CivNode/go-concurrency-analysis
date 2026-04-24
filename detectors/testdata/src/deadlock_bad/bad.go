package deadlocktest

import "sync"

var (
	m1 sync.Mutex
	m2 sync.Mutex
	m3 *sync.Mutex = &sync.Mutex{}
)

func bad() {
	go func() { // want "Deadlock"
		m1.Lock()
		m2.Lock()
		m2.Unlock()
		m1.Unlock()
	}()

	go func() { // want "Deadlock"
		m2.Lock()
		m1.Lock()
		m1.Unlock()
		m2.Unlock()
	}()
}

// Also flags through a *sync.Mutex pointer var.
func badPointer() {
	go func() { // want "Deadlock"
		m3.Lock()
		m1.Lock()
		m1.Unlock()
		m3.Unlock()
	}()
	go func() { // want "Deadlock"
		m1.Lock()
		m3.Lock()
		m3.Unlock()
		m1.Unlock()
	}()
}
