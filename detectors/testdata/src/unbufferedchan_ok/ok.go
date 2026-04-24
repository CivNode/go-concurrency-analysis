package unbufferedchantest

// Clean: send runs inside a goroutine; receive is in the caller.
func okPattern() int {
	ch := make(chan int)
	go func() {
		ch <- 42
	}()
	return <-ch
}

// Clean: buffered channel (explicit capacity).
func okBuffered() {
	ch := make(chan int, 1)
	ch <- 1
	_ = ch
}

// Clean: buffered with var-valued capacity still buffered (we only flag
// literal 0 or missing capacity).
func okBufferedVar() {
	n := 4
	ch := make(chan int, n)
	ch <- 1
	_ = ch
}

// Clean: send and range receive in the same function.
func okRange() {
	ch := make(chan int)
	go func() {
		for i := 0; i < 3; i++ {
			ch <- i
		}
		close(ch)
	}()
	for range ch {
	}
}

// Clean: explicit `<-ch` receive.
func okReceive() {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
	v := <-ch
	_ = v
}

// Clean: make of a non-channel type is ignored.
func okSlice() {
	s := make([]int, 0)
	_ = s
}
