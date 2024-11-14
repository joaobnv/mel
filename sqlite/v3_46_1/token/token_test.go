package token

import (
	"fmt"
	"testing"
)

func TestToken(t *testing.T) {
	tok := New([]byte("table"), KindTable)
	expected := "<table, Table>"
	if expected != tok.String() {
		fmt.Printf("expected %q, got %q", expected, tok.String())
	}

	k := kind(-1)
	tok = New(nil, &k)
	expected = "<-1>"
	if expected != tok.String() {
		fmt.Printf("expected %q, got %q", expected, tok.String())
	}
}
