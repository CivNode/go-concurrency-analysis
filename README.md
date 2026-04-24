# go-concurrency-analysis

Static detectors for the common Go concurrency mistakes: deadlocks, goroutine
leaks, unsynchronized shared state, defer inside loops, and unbuffered channel
sends with no paired receive. Part of the CivNode Training semantic engine.

This package is a complement to `go test -race`, not a replacement. It runs
without executing code, so it is cheap enough to run on every keystroke in a
kata editor. In exchange, each detector is deliberately conservative: the goal
is zero false positives on realistic code, accepting misses on shapes that
cannot be proven statically.

## Install

```bash
go get github.com/CivNode/go-concurrency-analysis@v0.1.0
```

## Quick start

```go
package main

import (
    "fmt"
    "os"

    ca "github.com/CivNode/go-concurrency-analysis"
)

func main() {
    src, _ := os.ReadFile("main.go")
    findings, err := ca.Analyze(src)
    if err != nil {
        panic(err)
    }
    for _, f := range findings {
        fmt.Printf("%s:%d: [%s] %s\n", f.Pos.Filename, f.Pos.Line, f.Kind, f.Message)
    }
}
```

To analyze a real module, call `AnalyzePackage` with go/packages patterns:

```go
findings, err := ca.AnalyzePackage("./...")
```

## Detectors (v0.1.0)

| Kind                     | What it flags                                                                      |
|--------------------------|------------------------------------------------------------------------------------|
| `Deadlock`               | Reciprocal `sync.Mutex.Lock` order on the same package-level mutex pair across two goroutines. |
| `GoroutineLeak`          | Inline `go func() { for { ... } }()` with no return, no `ctx.Done()` read, no channel operation, no `break`. |
| `UnsyncShared`           | Package-level variable written from two or more functions where at least one write runs inside a goroutine, with no mutex Lock / atomic op guarding it. |
| `DeferInLoop`            | `defer` statement whose nearest enclosing block is a for or range loop body. |
| `UnbufferedChanDeadlock` | Local `ch := make(chan T)` or `make(chan T, 0)` with a send and no matching receive visible in the same function (self-send pattern). |

Each detector lives in `detectors/<kind>.go` and exports an
`*analysis.Analyzer` whose `Name` matches the `Kind` string, so it can also be
plugged into `unitchecker`, `multichecker`, or `golangci-lint` plugins.

## Non-goals

- Whole-program flow analysis. We do not chase calls across function
  boundaries. Katas for these detectors keep the bug local to one function
  or one goroutine closure, which is where novice mistakes happen anyway.
- Races on local slice / map / struct fields. These are `go test -race`
  territory.

## Licence

Apache-2.0. See [LICENSE](./LICENSE).
