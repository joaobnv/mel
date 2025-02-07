package token

import (
	"fmt"
	"testing"
)

func TestToken(t *testing.T) {
	tok := New([]byte("."), kindDot)
	expected := `<".", Dot>`
	if expected != tok.String() {
		fmt.Printf("expected %q, got %q", expected, tok.String())
		t.Fail()
	}

	k := kind(-1)
	tok = New(nil, &k)
	expected = "<-1>"
	if expected != tok.String() {
		fmt.Printf("expected %q, got %q", expected, tok.String())
		t.Fail()
	}
}
