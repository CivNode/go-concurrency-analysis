package concurrencyanalysis

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestAnalyze_DeferInLoop verifies the top-level Analyze entrypoint finds a
// defer-in-loop in a single source buffer without a caller-supplied module.
func TestAnalyze_DeferInLoop(t *testing.T) {
	src := []byte(`package x

import "os"

func f(paths []string) {
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			continue
		}
		defer file.Close()
		_ = file
	}
}
`)
	findings, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !containsKind(findings, DeferInLoop) {
		t.Fatalf("expected DeferInLoop finding, got: %v", findings)
	}
}

// TestAnalyze_DeadlockAndChan mixes two detectors in one source.
func TestAnalyze_DeadlockAndChan(t *testing.T) {
	src := []byte(`package x

import "sync"

var (
	mA sync.Mutex
	mB sync.Mutex
)

func f() {
	go func() {
		mA.Lock()
		mB.Lock()
		mB.Unlock()
		mA.Unlock()
	}()
	go func() {
		mB.Lock()
		mA.Lock()
		mA.Unlock()
		mB.Unlock()
	}()
}

func g() {
	ch := make(chan int)
	ch <- 1
}
`)
	findings, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if !containsKind(findings, Deadlock) {
		t.Errorf("expected Deadlock finding, got: %v", kinds(findings))
	}
	if !containsKind(findings, UnbufferedChanDeadlock) {
		t.Errorf("expected UnbufferedChanDeadlock finding, got: %v", kinds(findings))
	}
}

func TestAnalyze_EmptySource(t *testing.T) {
	_, err := Analyze(nil)
	if err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestAnalyze_InvalidSource(t *testing.T) {
	_, err := Analyze([]byte("not valid go"))
	if err == nil {
		t.Fatal("expected error for invalid go source")
	}
}

// Kind.String sanity.
func TestKindString(t *testing.T) {
	for _, tc := range []struct {
		k    Kind
		want string
	}{
		{Deadlock, "Deadlock"},
		{GoroutineLeak, "GoroutineLeak"},
		{UnsyncShared, "UnsyncShared"},
		{DeferInLoop, "DeferInLoop"},
		{UnbufferedChanDeadlock, "UnbufferedChanDeadlock"},
		{Kind(-1), "Unknown"},
	} {
		if got := tc.k.String(); got != tc.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tc.k, got, tc.want)
		}
	}
}

func containsKind(fs []Finding, k Kind) bool {
	for _, f := range fs {
		if f.Kind == k {
			return true
		}
	}
	return false
}

func kinds(fs []Finding) string {
	parts := make([]string, len(fs))
	for i, f := range fs {
		parts[i] = fmt.Sprintf("%s@%s", f.Kind, f.Pos)
	}
	return strings.Join(parts, ", ")
}

var _ = errors.New // keep import stable for any future use
