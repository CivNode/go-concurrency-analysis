package concurrencyanalysis

import "go/token"

// Finding is a single static analysis report.
type Finding struct {
	Kind    Kind
	Message string
	Pos     token.Position
}
