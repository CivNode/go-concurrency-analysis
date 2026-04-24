package unsyncsharedtest

var counter int

func inc() {
	counter++ // want "UnsyncShared"
}

func asyncInc() {
	go func() {
		counter++ // want "UnsyncShared"
	}()
}

// Writes via plain assignment + compound assignment.
var total int

func set(n int) {
	total = n // want "UnsyncShared"
}

func accumulate() {
	go func() {
		total += 1 // want "UnsyncShared"
	}()
}
