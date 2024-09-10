// package lexer deals with the lexical scanning.
package lexer

// Lexer is a lexical scanner
type Lexer struct {
	code []byte
}

// New creates a new Lexer that reads from code.
func New(code string) *Lexer {
	return &Lexer{code: []byte(code)}
}
