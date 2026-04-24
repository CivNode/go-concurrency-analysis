package concurrencyanalysis

// Kind classifies a concurrency finding.
type Kind int

// Kind values. New detectors append a new constant.
const (
	// Deadlock flags a reciprocal sync.Mutex lock order between two goroutines.
	Deadlock Kind = iota + 1
	// GoroutineLeak flags a goroutine body that has no exit signal in sight.
	GoroutineLeak
	// UnsyncShared flags a package level variable written from multiple
	// functions without synchronization, at least one of which is spawned
	// in a goroutine.
	UnsyncShared
	// DeferInLoop flags a defer statement inside a for loop body.
	DeferInLoop
	// UnbufferedChanDeadlock flags an unbuffered channel send with no
	// matching receive visible in the same function.
	UnbufferedChanDeadlock
)

// String returns the canonical name used by analysistest `// want "..."` comments.
func (k Kind) String() string {
	switch k {
	case Deadlock:
		return "Deadlock"
	case GoroutineLeak:
		return "GoroutineLeak"
	case UnsyncShared:
		return "UnsyncShared"
	case DeferInLoop:
		return "DeferInLoop"
	case UnbufferedChanDeadlock:
		return "UnbufferedChanDeadlock"
	default:
		return "Unknown"
	}
}
