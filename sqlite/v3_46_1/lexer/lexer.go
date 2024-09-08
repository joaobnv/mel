package lexer

// Lexer is a lexical scanner
type Lexer struct {
	code string
}

// New creates a new Lexer that reads from code.
func New(code string) *Lexer {
	return &Lexer{code: code}
}
