// Package concurrencyanalysis is part of the CivNode Training semantic engine.
// It hosts a small set of conservative static detectors for the common Go
// concurrency mistakes Training katas target: deadlocks, leaked goroutines,
// unsynchronized shared state, defers inside loops, and unbuffered channel
// sends with no paired receive.
//
// Use Analyze for a single source file loaded in memory, or AnalyzePackage
// for one or more go/packages patterns.
//
// See https://github.com/CivNode/go-concurrency-analysis for details.
package concurrencyanalysis
