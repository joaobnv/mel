// package lexer deals with the lexical scanning.
package lexer

import (
	"unicode/utf8"
)

// Lexer is a lexical scanner
type Lexer struct {
	// r is the reader that the lexer uses for reading the runes from the code.
	r *reader
}

// New creates a new Lexer that reads from code.
func New(code []byte) *Lexer {
	return &Lexer{r: newReader(code)}
}

// reader reads from the code.
type reader struct {
	// code is the code to be read.
	code []byte
	// offset is the current offset on code.
	offset int64
}

// newReader creates a new reader that reads from code.
func newReader(code []byte) *reader {
	return &reader{
		code: code,
	}
}

// readRune reads the next rune from the code. It panics on error.
func (r *reader) readRune() (rn rune, eof bool) {
	rn, size := utf8.DecodeRune(r.code[r.offset:])
	if rn == utf8.RuneError {
		if size == 0 {
			return 0, true
		} else {
			panic("utf-8 encoding invalid")
		}
	}
	r.offset += int64(size)
	return
}

// unreadRune seek to the start of the rune before the current offset. If the current or resulting offset is at
// the start of the code then onStart will be true.
func (r *reader) unreadRune() (onStart bool) {
	if r.offset == 0 {
		return true
	}
	for i := r.offset - 1; i >= 0; i-- {
		if utf8.RuneStart(r.code[i]) {
			r.offset = i
			return r.offset == 0
		}
	}
	panic("utf-8 encoding invalid")
}
