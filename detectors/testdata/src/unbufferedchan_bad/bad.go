package unbufferedchantest

func bad() {
	ch := make(chan int)
	ch <- 1 // want "UnbufferedChanDeadlock"
}

func badSecond() {
	done := make(chan struct{})
	done <- struct{}{} // want "UnbufferedChanDeadlock"
}

// Explicit capacity 0 is still unbuffered.
func badZero() {
	ch := make(chan int, 0)
	ch <- 1 // want "UnbufferedChanDeadlock"
}
