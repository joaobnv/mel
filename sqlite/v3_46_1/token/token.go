// This package deals with tokens.
package token

import (
	"strconv"
	"strings"
)

// Token is a token from the code.
type Token struct {
	// Lexeme is the lexeme of the token.
	Lexeme []byte
	// Kind is the kind of the token.
	Kind Kind
}

// New creates a Token.
func New(lexeme []byte, kind Kind) *Token {
	return &Token{Lexeme: lexeme, Kind: kind}
}

// String returns a string representation of t.
func (t *Token) String() string {
	var b strings.Builder
	b.WriteRune('<')
	if t.Lexeme != nil {
		b.WriteString(strconv.Quote(string(t.Lexeme)))
		b.WriteString(", ")
	}
	b.WriteString(t.Kind.String())
	b.WriteRune('>')
	return b.String()
}
