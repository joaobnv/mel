// This package generates strings.
package sqlgen

import (
	"iter"
	"math/rand/v2"
)

type SyntaxConfig struct {
	// Rand is used to make decisions such as the number of turns in a cycle.
	Rand               *rand.Rand
	TurnsInCycleLimit  uint8 // the maximum number of turns in cycles. Default 2.
	PossibilitiesLimit uint8 // the maximum number of generated results by constructs that can generate more than one. Default 3.
}

// Syntax generates a sequence of strings that respect the syntax of the SQLite dialect of SQL.
func Syntax() iter.Seq[string] {
	s := newSyntax()
	cfg := &SyntaxConfig{
		Rand:              rand.New(rand.NewPCG(0, 0)),
		TurnsInCycleLimit: 2, PossibilitiesLimit: 3,
	}
	return (*s.sqlStmt()).gen(nil, cfg, s)
}
